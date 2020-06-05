from pathlib import Path
from typing import Union, List

from notion.client import NotionClient
from notion.block import PageBlock
from notion.collection import Collection, NotionDate

from .config import Config
from .templates import load_template
from .page import page_path, front_matter


def notion_client(token_v2: str) -> NotionClient:
    config = Config(token_v2=token_v2)
    client = NotionClient(token_v2=config.token_v2)
    return client


def notion_blog_database(client: NotionClient, db_url: str) -> Collection:
    """
    Returns a Notion database, wraped into a `notion.Collection` for
    easy access to it's rows.
    """
    return client.get_collection_view(db_url).collection


def all_pages(blog: Collection) -> List[PageBlock]:
    "Returns all Notion PageBlocks"
    return blog.get_rows()


def published_pages(blog: Collection) -> List[PageBlock]:
    "Returns the list of published pages."
    # FIXME: This needs to be a filtered query instead.
    return [post for post in blog.get_rows() if post.get_all_properties()["published"]]


def draft_pages(blog: Collection) -> List[PageBlock]:
    "Returns the list of draft pages."
    # FIXME: This needs to be a filtered query instead.
    return [
        post for post in blog.get_rows() if not post.get_all_properties()["published"]
    ]


def page_to_markdown(page: PageBlock, config: Config) -> str:
    "Translates a Notion Page (`PageBlock`) into a Markdown string and returns it."
    return load_template("post", debug=True, config=config).render(
        page=page, front_matter=front_matter(page, config)
    )
