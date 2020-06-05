from pathlib import Path
from string import Template
from datetime import datetime

import re

from notion.collection import NotionDate, CollectionRowBlock
from notion.block import PageBlock

from .config import Config

"""
Functions that operate on Notion's `PageBlock`.
"""


def page_path(page: PageBlock, dest_dir: Path = Path(".")) -> Path:
    "Build a .md file path in `dest_dir` based on a Notion page metadata."
    fname = __title_to_slug(page.title) + ".md"
    return dest_dir / fname


def __title_to_slug(title: str) -> str:
    return re.sub(r"[^\w\-]", "", re.sub(r"\s+", "-", title)).lower()


def front_matter(page: CollectionRowBlock, config: Config) -> str:
    "Builds and returns a page front matter in a yaml-like format."
    # Start with all the properties from the page, including formulas and rollups.
    all_props = page.get_all_properties()

    # Add the default layout from the config
    # if there's no layout property on the page itself.
    if "layout" not in all_props:
        all_props["layout"] = config.default_layout

    # Add default published_at if there's no specific property for it.
    if "published_at" not in all_props:
        last_edited_time = datetime.utcfromtimestamp(
            int(page.get("last_edited_time")) / 1000
        )
        all_props["published_at"] = last_edited_time

    # Select only properties that are not empty
    renderables = {k: v for k, v in all_props.items() if v != ""}
    return __sanitize_front_matter(renderables)


def __sanitize_front_matter(items: dict) -> dict:
    "Sanitizes and returns front matter items as a dictionary."
    for k, v in items.items():
        if type(v) not in [str, list]:
            if isinstance(v, NotionDate):
                items[k] = v.start
            if isinstance(v, bool):
                items[k] = str(v).lower()
    return items


def page_url_substitutions(page: PageBlock, config: Config) -> dict:
    "Builds and returns a `dict` of substitutions to build URL to this page in the Blog."
    # Start with a dict from `front_m   subs = front_matter(page, config)
    subs = front_matter(page, config)

    # Grab the config baseurl
    subs["baseurl"] = config["baseurl"]

    # FIXME Clean this up into a mapping of callables?
    #
    if "title" in subs:
        subs["title"] = __title_to_slug(subs["title"])

    if "categories" in subs:
        subs["categories"] = "/".join(subs["categories"])
    else:
        subs["categories"] = ""

    if "published_at" in subs:
        subs["year"], subs["month"], subs["day"], *_ = subs["published_at"].timetuple()

    return subs


def build_page_url(page: PageBlock, pattern: Template, config: Config) -> str:
    "Build the URL string for a given URL pattern and returns it as `str`."
    return pattern.substitute(page_url_substitutions(page, config))
