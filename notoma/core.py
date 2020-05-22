import yaml
from pathlib import Path
from typing import Union

from notion.client import NotionClient
from notion.collection import Collection
from notion.block import *

from .config import Config
from .templates import template


def notion_client(token_v2: str) -> NotionClient:
    config = Config(token_v2=token_v2)
    client = NotionClient(token_v2=config.token_v2)
    return client


def notion_blog_database(client: NotionClient, db_url: str) -> Collection:
    """Returns a Notion database, wraped into a `notion.Collection` for
     easy access to it's rows."""
    return client.get_collection_view(db_url).collection


def page2md(page: PageBlock) -> str:
    """Translates a Notion Page (`PageBlock`) into a Markdown string."""
    return template("post", debug=True).render(
        page=page, front_matter=front_matter(page)
    )


def page2path(page: PageBlock, dest_dir: Path = Path(".")) -> Path:
    """Build a .md file path in `dest_dir` based on a Notion page metadata."""
    fname = "-".join(page.title.lower().replace(".", "").split(" ")) + ".md"
    return dest_dir / fname


def front_matter(page: PageBlock) -> str:
    """Builds a page front matter in a yaml-like format."""
    internals = ["published", "title"]
    all_props = page.get_all_properties()
    renderables = {k: v for k, v in all_props.items() if k not in internals}

    # TODO:
    # This needs a way to process bool values and dates
    # And do other sanity checks.
    # return {k: v for k, v in renderables.items if type(v) in [str, bool, list]}
    return renderables


def notion2md(token_v2: str, database_url: str, dest: Union[str, Path]) -> None:
    """
    Grab Notion Blog database using auth token `token_v2`,
    convert posts in database `database_url` to Markdown,
    and save them to `dest`.
    """

    client = notion_client(token_v2)
    database = notion_blog_database(client, database_url)

    for post in database.get_rows():
        page2path(post, dest_dir=dest).write_text(page2md(post))
