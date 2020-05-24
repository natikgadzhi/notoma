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

"""
Provides templates that are used in Notion -> Markdown conversion.
They're currently not utilized, as the render engine actually uses
`notion.markdown.notion_to_markdown` instead
"""


def template(name: Union[str, Path], debug: bool = False) -> Template:
    """
    Loads the template file `templates/{name}.md.j2`.
    """

    filters = dict(
        template_name=__template_name,
        render_block=__render_block,
        numbered_list_index=__numbered_list_index,
        notion_url=__notion_url,
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

    return env.get_template(f"{name}.md.j2")


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
    "Returns snake_cased block name."
    return __snake_case(block.__class__.__name__)[:-6]


def __snake_case(input_string: str) -> str:
    "Returns a new string containing the input string, in camel_case."
    output = re.sub("(.)([A-Z][a-z]+)", r"\1_\2", input_string)
    return re.sub("([a-z0-9])([A-Z])", r"\1_\2", output).lower()


def __notion_url(page: block.PageBlock) -> str:
    "Returns the page's Notion URL."
    return page.get_browseable_url()
