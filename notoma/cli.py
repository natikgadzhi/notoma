import os
import click
from .config import Config

"""
`cli` Module only has thin wrappers around Notoma Python API
that invokes the API with provided configuration.

Run `notoma` for the help article on how to use the CLI.
"""


# The CLI runner
#
@click.group(
    help="""Build your staticg gen blog with Notion.
    Notoma converts Notion database of blog posts into a directory of .md files."""
)
@click.option(
    "--token_v2", "-t", type=str, help="Notion auth token from the cookie.",
)
def runner(token_v2, auto_envvar_prefix="NOTOMA_NOTION") -> None:
    pass


@runner.command(help="Convert Notion Blog to Markdown files.")
@click.option(
    "--blog", "-b", type=str, help="Notion blog URL.",
)
@click.option("--out", "-o", default="posts", help="Path to put .md files.")
def convert(out: str, token_v2: str = None, blog: str = None) -> None:
    """
    Convert Notion Blog pages to Markdown posts once
    """
    config = Config()

    print(config)


@runner.command()
def watch() -> None:
    """
    Watch for updates in the Notion Blog and update the Markdown posts
    """
    pass


@runner.command()
def new() -> None:
    """
    Create a new Notion Blog
    """
    pass
