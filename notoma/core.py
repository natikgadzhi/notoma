import yaml
from pathlib import Path
from typing import Union, List

from notion.client import NotionClient
from notion.block import PageBlock
from notion.collection import Collection, NotionDate

from .config import Config
from .templates import template


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


def page_to_markdown(page: PageBlock) -> str:
    "Translates a Notion Page (`PageBlock`) into a Markdown string."
    return template("post", debug=True).render(
        page=page, front_matter=front_matter(page)
    )


def page_path(page: PageBlock, dest_dir: Path = Path(".")) -> Path:
    "Build a .md file path in `dest_dir` based on a Notion page metadata."
    fname = "-".join(page.title.lower().replace(".", "").split(" ")) + ".md"
    return dest_dir / fname


def front_matter(page: PageBlock, config: Config = Config()) -> str:
    "Builds and returns a page front matter in a yaml-like format."
    internals = ["title"]
    all_props = page.get_all_properties()
    if "layout" not in all_props:
        all_props["layout"] = config.default_layout
    renderables = {k: v for k, v in all_props.items() if k not in internals}
    return __sanitize_front_matter(renderables)


def __sanitize_front_matter(items: dict) -> dict:
    "Sanitizes and returns front matter items."
    for k, v in items.items():
        if type(v) not in [str, list]:
            if isinstance(v, NotionDate):
                items[k] = v.start
            if isinstance(v, bool):
                items[k] = str(v).lower()
    return items
