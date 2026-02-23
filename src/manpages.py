# Copyright 2025 Canonical
# See LICENSE file for licensing details.

"""Representation of the manpages service."""

import json
import logging
import os
import re
import shutil
from dataclasses import asdict, dataclass, field
from pathlib import Path
from subprocess import CalledProcessError

import charms.operator_libs_linux.v0.apt as apt
from charms.operator_libs_linux.v0.apt import PackageError, PackageNotFoundError
from charms.operator_libs_linux.v1.systemd import service_restart, service_running
from jinja2 import DictLoader, Environment

logger = logging.getLogger(__name__)

# Used to fetch release codenames from the config string passed to the charm.
RELEASES_PATTERN = re.compile(r"([a-z]+)(?:[,][ ]*)*")

# Directories required by the manpages charm.
APP_DIR = Path("/app")
WWW_DIR = APP_DIR / "www"
BIN_DIR = APP_DIR / "bin"

# Configuration files created by the manpages charm.
CONFIG_PATH = WWW_DIR / "config.json"

INGEST_SERVICE_PATH = "/etc/systemd/system/ingest-manpages.service"
INGEST_SERVICE_TEMPLATE = """
[Unit]
Description=Ingest Manpages into the Repository

[Service]
Type=simple
ExecStart=/app/bin/ingest -config={{ config_path }}
{{ proxy_config }}
"""

SERVER_SERVICE_PATH = "/etc/systemd/system/manpages.service"
SERVER_TEMPLATE = """
[Unit]
Description=Manpages Application Server

[Service]
Type=simple
ExecStart=/app/bin/server -config={{ config_path }}
{{ proxy_config }}
"""


@dataclass
class ManpagesConfig:
    """Configuration for manpages service."""

    site: str = "http://manpages.ubuntu.com"
    archive: str = "http://archive.ubuntu.com/ubuntu"
    public_html_dir: str = str(WWW_DIR)
    releases: dict = field(
        default_factory=lambda: {
            "jammy": "22.04",
            "noble": "24.04",
            "oracular": "24.10",
            "plucky": "25.04",
            "questing": "25.10",
        }
    )
    repos: list = field(default_factory=lambda: ["main", "restricted", "universe", "multiverse"])
    arch: str = "amd64"


class Manpages:
    """Represent a manpages instance in the workload."""

    def __init__(self, launchpad_client):
        self.launchpad_client = launchpad_client

    def install(self):
        """Install manpages."""
        try:
            apt.update()
        except CalledProcessError as e:
            logger.error("failed to update package cache: %s", e)
            raise

        try:
            apt.add_package("mandoc")
        except PackageNotFoundError:
            logger.error("failed to find package mandoc in package cache")
            raise
        except PackageError as e:
            logger.error("failed to install mandoc: %s", e)
            raise

        # Get path to charm source, inside which is the app and its configuration.
        source_path = Path(__file__).parent.parent

        # Install the binaries and systemd units.
        APP_DIR.mkdir(parents=True, exist_ok=True)
        WWW_DIR.mkdir(parents=True, exist_ok=True)
        shutil.copytree(source_path / "bin", BIN_DIR, dirs_exist_ok=True)
        self._template_systemd_units()

    def configure(self, releases: str, url: str):
        """Configure the manpages service."""
        try:
            config = self._build_config(releases, url)
        except ValueError as e:
            logger.error("failed to build manpages configuration: invalid releases spec: %s", e)
            raise

        # Ensure the systemd unit is updated in case the Juju proxy config has changed.
        self._template_systemd_units()

        # Write the configuration file for the application.
        with open(CONFIG_PATH, "w") as f:
            json.dump(asdict(config), f)

    def restart(self):
        """Restart the manpages services."""
        try:
            service_restart("manpages")
        except CalledProcessError as e:
            logger.error("failed to restart manpages services: %s", e)
            raise

    def update_manpages(self):
        """Update the manpages."""
        try:
            service_restart("ingest-manpages")
            self.purge_unused_manpages()
        except CalledProcessError as e:
            logger.error("failed to update manpages: %s", e)
            raise

    def purge_unused_manpages(self):
        """Purge unused manpages.

        If a release is no longer configured in the application config, but
        previously was, this function removes the manpages for that release.
        """
        # No releases have yet been downloaded, skip this step
        if not (WWW_DIR / "manpages").exists():
            return

        with open(CONFIG_PATH, "r") as f:
            config = json.load(f)

        configured_releases = config["releases"].keys()
        releases_on_disk = [f.name for f in os.scandir(WWW_DIR / "manpages") if f.is_dir()]

        for release in releases_on_disk:
            if release not in configured_releases:
                logger.info("purging manpages for '%s'", release)
                shutil.rmtree(WWW_DIR / "manpages" / release)

    @property
    def updating(self) -> bool:
        """Report whether the manpages are currently being updated."""
        return service_running("ingest-manpages")

    def _build_config(self, releases: str, url: str) -> ManpagesConfig:
        """Build a ManpagesConfig object using a set of specified release codenames."""
        releases_list = RELEASES_PATTERN.findall(releases)
        if not releases_list:
            raise ValueError("failed to build manpages config: invalid releases specified")

        config = ManpagesConfig()
        config.site = url

        # Get the release map for the specified release codenames.
        try:
            config.releases = self.launchpad_client.release_map(releases_list)
        except ValueError as e:
            logger.error("failed to build manpages config: %s", e)
            raise ValueError(f"failed to build manpages config: {e}")

        return config

    def _template_systemd_units(self):
        """Template out systemd unit file including proxy variables."""
        # Maps Juju specific proxy environment variables to system equivalents.
        proxy_vars = [
            ("JUJU_CHARM_HTTP_PROXY", "HTTP_PROXY"),
            ("JUJU_CHARM_HTTPS_PROXY", "HTTPS_PROXY"),
            ("JUJU_CHARM_NO_PROXY", "NO_PROXY"),
        ]

        # Iterate over the possible proxy variables, and if a value is set,
        # construct a systemd 'Environment' line and add to the list of lines.
        # Add both upper and lower case to account for expectations of different applications
        # e.g. curl only accepts the lower case form of "http_proxy"
        # https://everything.curl.dev/usingcurl/proxies/env.html#http_proxy-in-lower-case-only
        lines = []
        for v in proxy_vars:
            if proxy := os.environ.get(v[0], None):
                lines.append(f"\nEnvironment={v[1]}={proxy}")
                lines.append(f"\nEnvironment={v[1].lower()}={proxy}")

        # Template out the unit file and write it to disk.
        env = Environment(
            loader=DictLoader(
                {
                    "ingest-manpages.service.j2": INGEST_SERVICE_TEMPLATE,
                    "manpages.service.j2": SERVER_TEMPLATE,
                }
            )
        )
        context = {"proxy_config": "\n".join(lines), "config_path": CONFIG_PATH}

        template = env.get_template("ingest-manpages.service.j2")
        with open(INGEST_SERVICE_PATH, "w") as f:
            f.write(template.render(context))

        template = env.get_template("manpages.service.j2")
        with open(SERVER_SERVICE_PATH, "w") as f:
            f.write(template.render(context))
