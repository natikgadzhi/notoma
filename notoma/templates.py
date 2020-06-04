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
import re
import string

from .config import Config
from .page import front_matter

"""
Provides templates that are used in Notion -> Markdown conversion.
They're currently not utilized, as the render engine actually uses
`notion.markdown.notion_to_markdown` instead
"""


def template(
    name: Union[str, Path], debug: bool = False, config: Config = Config()
) -> Template:
    """
    Loads the template file `templates/{name}.md.j2`.
    """

    filters = dict(
        template_name=__template_name,
        render_block=__render_block,
        numbered_list_index=__numbered_list_index,
        notion_url=__notion_url,
        linked_page_id=__linked_page_id,
        linked_page_url=__linked_page_url,
        linked_page_title=__linked_page_title,
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


def __linked_page_id(b: block.TextBlock) -> str:
    "Jinja filter. Returns a linked Notion page ID from a Notion Block."
    # This assumes that the block has a raw formatting in the `properties.title`
    # path, and returns the appropriate field.
    # If and when the Notion API changes, this will fail.
    return b.get("properties.title")[0][1][0][1]


def __get_linked_page(ctx: Context, page_id: str) -> block.PageBlock:
    "Returns the linked PageBlock by it's block id. Not exposed to the template."
    this_page = ctx["page"]
    linked_page = block.PageBlock(this_page._client, page_id)

    if linked_page.parent == this_page.parent:
        return linked_page
    else:
        raise ValueError(f"The page {page_id} has different parent.")


@contextfilter
def __linked_page_url(ctx: Context, b: block.TextBlock) -> str:
    "Jinja filter, returns a URL of a linked page."
    # TODO: Use a helper provided in .core that will build the page URL with
    # the configurable permalinks pattern.
    #

    config = ctx["config"]
    linked_page = __get_linked_page(ctx, __linked_page_id(b))

    pattern = string.Template(config["permalink_pattern"])
    if pattern is None:
        return linked_page.get_browseable_url()

    subs = front_matter(ctx["page"], config)

    subs["baseurl"] = config["baseurl"]

    # The link needs default values for when there's no
    # published_at date or no categories

    return pattern.substitute(subs)


def __page_link_subs(page: block.PageBlock, config: Config, fm) -> dict:
    return {**front_matter, "baseurl": config["baseurl"]}


@contextfilter
def __linked_page_title(ctx: Context, b: block.TextBlock) -> str:
    "Jinja filter, returns the title of the linked page."
    return __get_linked_page(ctx, __linked_page_id(b)).title


# Jinja Tests
#
def __is_notion_link(b: block.TextBlock) -> bool:
    "Jinja test. Returns true if the block raw title property has a Notion link in it."
    if b.title == "" or b.title is None:
        return False

    raw_title = b.get("properties.title")
    if len(raw_title[0]) > 1:
        modifier = raw_title[0][1]
        # the item after the modifier "p" is the linked page ID
        return modifier[0][0] == "p"
