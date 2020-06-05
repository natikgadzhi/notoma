from typing import Union
from pathlib import Path

from jinja2 import (
    Environment,
    Template,
    PackageLoader,
    select_autoescape,
    contextfilter,
)
from jinja2.runtime import Context

import notion.block as block
from notion.collection import CollectionRowBlock
from notion.markdown import notion_to_markdown

import re
import string

from .config import Config
from .page import build_page_url

"""
Provides templates that are used in Notion -> Markdown conversion.
They're currently not utilized, as the render engine actually uses
`notion.markdown.notion_to_markdown` instead
"""


def load_template(
    name: Union[str, Path], debug: bool = False, config: Config = Config()
) -> Template:
    """
    Loads the template file `templates/{name}.md.j2`.
    """

    filters = dict(
        template_name=__template_name,
        render_block=__render_block,
        snake_case=__snake_case,
        numbered_list_index=__numbered_list_index,
        notion_url=__notion_url,
        preprocess_notion_links=__preprocess_notion_links,
    )

    env = Environment(
        loader=PackageLoader("notoma", "templates"),
        autoescape=select_autoescape(["md"]),
        trim_blocks=True,
        lstrip_blocks=True,
    )

    for fl, func in filters.items():
        env.filters[fl] = func

    env.globals["debug"] = debug
    env.globals["config"] = config
    env.tests["notion_link"] = __is_notion_link
    return env.get_template(f"{name}.md.j2")


#
# Helpers
#


def __get_linked_page(this_page: block.PageBlock, page_id: str) -> CollectionRowBlock:
    "Returns the linked CollectionRowBlock by it's block id. Not exposed to the template."
    linked_page = CollectionRowBlock(this_page._client, page_id)

    if not linked_page.parent == this_page.parent:
        raise ValueError(f"The page {page_id} has different parent.")

    return linked_page


def __linked_page_url(config: Config, page: CollectionRowBlock) -> str:
    "Returns a URL of a linked page."
    pattern = string.Template(config["permalink_pattern"])

    # TODO: Add tests and remove this quite questionable behavior.
    # Or make it work on a flag, and add logging.
    if pattern is None:
        return page.get_browseable_url()

    return build_page_url(page, pattern, config)


#
# Jinja filters
#


def __template_name(block: block.Block) -> str:
    """
    Finds a Jinja template name for the provided Notion `Block`,
    checks that the template exists, and returns the template name as `str`.
    """
    template_name = f"blocks/_{__block_type(block)}.md.j2"
    template_path = Path(__file__).parent / f"templates/{template_name}"

    if template_path.exists():
        return template_name
    else:
        return None


@contextfilter
def __render_block(ctx: Context, block: block.Block) -> str:
    """
    Jinja filter that wraps the block in it's markdown equivalent if possible,
    or returns it's title.
    """
    if ctx["debug"]:
        print(f"Unsupported block type: {__block_type(block)} in {ctx['page'].title}.")
    return str(block.title)


@contextfilter
def __numbered_list_index(ctx: Context, this: block.Block) -> str:
    "Traverses the parent PageBlock and calculates the correct index for the numbered list item block."
    index = 1
    for item in ctx["page"].children:
        if isinstance(item, block.NumberedListBlock):
            if item.id == this.id:
                return index
            else:
                index += 1
        else:
            index = 1
    raise ValueError(
        f"Expected the page {ctx['page'].id} to include the provided block {this.id}"
    )


def __block_type(block: block.Block) -> str:
    "Jinja filter. Returns snake_cased block name."
    return __snake_case(block.__class__.__name__)[:-6]


def __snake_case(input_string: str) -> str:
    "Jinja filter. Returns a new string containing the input string, in camel_case."
    output = re.sub("(.)([A-Z][a-z]+)", r"\1_\2", input_string)
    return re.sub("([a-z0-9])([A-Z])", r"\1_\2", output).lower()


def __notion_url(page: block.PageBlock) -> str:
    "Jinja filter. Returns the page's (or blocks) Notion URL."
    return page.get_browseable_url()


@contextfilter
def __preprocess_notion_links(ctx: Context, b: block.TextBlock) -> str:
    config = ctx["config"]
    chunks = b.get("properties.title")

    if chunks is None or len(chunks) == 0:
        return ""

    for chunk in chunks:
        # For the chunks that have formatting
        if len(chunk) > 1:
            # Each chunk can have multiple modifiers
            for modifier in chunk[1]:
                # Only process Notion page links.
                # Replace Notion link formatting with a regular
                # Markdown link formatting and invoke Noton Py's
                # formatting function.
                if modifier[0] == "p":
                    page_id = modifier[1]
                    page = __get_linked_page(ctx["page"], page_id)
                    modifier[0] = "a"
                    modifier[1] = __linked_page_url(config, page)
                    chunk[0] = page.title

    return notion_to_markdown(chunks)


#
# Jinja Tests
#


def __is_notion_link(b: block.TextBlock) -> bool:
    "Jinja test. Returns true if the block raw title property has a Notion link in it."
    if b.title == "" or b.title is None:
        return False

    raw_title = b.get("properties.title")
    for chunk in raw_title:
        if len(chunk) > 1:
            modifier = chunk[1]
            return modifier[0][0] == "p"
