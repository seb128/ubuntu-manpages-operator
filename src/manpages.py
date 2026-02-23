# Copyright 2025 Canonical
# See LICENSE file for licensing details.

"""Representation of the manpages service."""

import json
import logging
import os
import re
from dataclasses import asdict, dataclass, field
from pathlib import Path

import ops
from ops.pebble import APIError, ConnectionError, PathError, ProtocolError

logger = logging.getLogger(__name__)

# Used to fetch release codenames from the config string passed to the charm.
RELEASES_PATTERN = re.compile(r"([a-z]+)(?:[,][ ]*)*")
WWW_DIR = Path("/app") / "www"
CONFIG_PATH = WWW_DIR / "config.json"
PORT = 8080


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

    def __init__(self, launchpad_client, container: ops.Container):
        self.launchpad_client = launchpad_client
        self.container = container

    def pebble_layer(self) -> ops.pebble.Layer:
        """Return a Pebble layer for managing manpages server and ingestion."""
        proxy_env = {
            "HTTP_PROXY": os.environ.get("JUJU_CHARM_HTTP_PROXY", ""),
            "HTTPS_PROXY": os.environ.get("JUJU_CHARM_HTTPS_PROXY", ""),
            "NO_PROXY": os.environ.get("JUJU_CHARM_NO_PROXY", ""),
        }
        return ops.pebble.Layer(
            {
                "services": {
                    "manpages": {
                        "override": "replace",
                        "summary": "manpages server",
                        "command": "/usr/bin/server -config=/app/www/config.json",
                        "startup": "enabled",
                    },
                    "ingest": {
                        "override": "replace",
                        "summary": "manpages ingestion",
                        "command": "/usr/bin/ingest -config=/app/www/config.json",
                        "startup": "enabled",
                        "on-success": "ignore",
                        "environment": proxy_env,
                    },
                },
                "checks": {
                    "up": {
                        "override": "replace",
                        "level": "alive",
                        "period": "30s",
                        "tcp": {"port": PORT},
                        "startup": "enabled",
                    },
                },
            }
        )

    def configure(self, releases: str, url: str):
        """Configure the manpages service."""
        try:
            config = self._build_config(releases, url)
        except ValueError as e:
            logger.error("failed to build manpages configuration: invalid releases spec: %s", e)
            raise

        # Write the configuration file for the application.
        try:
            self.container.push(CONFIG_PATH, json.dumps(asdict(config)), make_dirs=True)
        except (ProtocolError, ConnectionError, PathError, APIError) as e:
            logger.error("failed to push manpages configuration to container: %s", e)
            raise

    def update_manpages(self):
        """Update the manpages."""
        try:
            self.container.restart("ingest")
            self.purge_unused_manpages()
        except (ProtocolError, ConnectionError, APIError) as e:
            logger.error("failed to ingest manpages: %s", e)
            raise

    def purge_unused_manpages(self):
        """Purge unused manpages.

        If a release is no longer configured in the application config, but
        previously was, this function removes the manpages for that release.
        """
        # No releases have yet been downloaded, skip this step
        try:
            if not self.container.exists(WWW_DIR / "manpages"):
                return
        except (ProtocolError, ConnectionError, APIError) as e:
            logger.error("failed to check existence of manpages directory: %s", e)
            raise

        try:
            config = self.container.pull(str(CONFIG_PATH)).read()
        except (ProtocolError, ConnectionError, APIError) as e:
            logger.error("failed to pull manpages configuration: %s", e)
            raise

        config = json.loads(config)
        configured_releases = config["releases"].keys()

        try:
            files = self.container.list_files(WWW_DIR / "manpages")
        except (ProtocolError, ConnectionError, PathError, APIError) as e:
            logger.error("failed to list manpages directory: %s", e)
            raise

        releases_on_disk = [f.name for f in files if f.type == ops.pebble.FileType.DIRECTORY]

        for release in releases_on_disk:
            if release in configured_releases:
                continue

            logger.info("purging manpages for '%s'", release)
            try:
                self.container.remove_path(WWW_DIR / "manpages" / release)
            except (ProtocolError, ConnectionError, PathError, APIError) as e:
                logger.error("failed to remove manpages for '%s': %s", release, e)
                raise

    @property
    def updating(self) -> bool:
        """Report whether the manpages are currently being updated."""
        try:
            running = self.container.get_service("ingest").is_running()
            return running
        except (ProtocolError, ConnectionError, APIError, ops.ModelError) as e:
            logger.error("failed to get manpages ingest service status: %s", e)
            return False

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
