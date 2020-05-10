import yaml
from pathlib import Path
from typing import Union

from notion.client import NotionClient
from notion.collection import Collection
from notion.block import *

from notion.markdown import notion_to_markdown

from .config import Config


def notion_client(token_v2: str) -> NotionClient:
    config = Config(token_v2=token_v2)
    client = NotionClient(token_v2=config.token_v2)
    return client


def notion_blog_database(client: NotionClient, db_url: str) -> Collection:
    """Returns a Notion database, wraped into a `notion.Collection` for
     easy access to it's rows."""
    return client.get_collection_view(db_url).collection


def block2md(block: Block, counter:int = 1) -> str:
    """Transforms a Notion Block into a Markdown string."""

    if isinstance(block, TextBlock):
        return block.title

    elif isinstance(block, HeaderBlock):
        return f"# {block.title}"

    elif isinstance(block, SubheaderBlock):
        return f"## {block.title}"

    elif isinstance(block, SubsubheaderBlock):
        return f"### {block.title}"

    elif isinstance(block, QuoteBlock):
        return f"> {block.title}"

    elif isinstance(block, BulletedListBlock):
        return f"- {block.title}"

    elif isinstance(block, NumberedListBlock):
        return f"{counter}. {block.title}"

    elif isinstance(block, CodeBlock):
        return f"""
```{block.language}
{block.title}
```
"""

    elif isinstance(block, CalloutBlock):
        return f"> {block.icon} {block.title}"

    elif isinstance(block, DividerBlock):
        return "\n"
    else:
        return ""


def page2md(page: PageBlock) -> str:
    """Translates a Notion Page (`PageBlock`) into a Markdown string."""
    blocks = list()

    # Numbered lists iterator
    counter = 1

    for block in page.children:
        blocks.append(block2md(block, counter))

        if isinstance(block, NumberedListBlock):
            counter += 1
        else:
            counter = 1

    return page_front_matter(page) + "\n".join(blocks)


def page2path(page: PageBlock, dest_dir: Path = Path(".")) -> Path:
    """Build a .md file path in `dest_dir` based on a Notion page metadata."""
    return dest_dir/Path("-".join(page.title.lower().replace(".", "").split(" "))+ ".md")


def page_front_matter(page: PageBlock) -> str:
    """Builds a page front matter in a yaml-like format."""
    internals = ['published', 'title']
    renderables = { k:v for k,v in page.get_all_properties().items() if k not in internals }

    return f"""
---
{yaml.dump(renderables)}
---\n
"""


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