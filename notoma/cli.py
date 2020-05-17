import os
from pathlib import Path
import click

from .config import Config
from .core import (
    notion_client,
    notion_blog_database,
    page_front_matter,
    page2md,
    page2path,
)
from .version import __version__

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
def runner() -> None:
    pass


@runner.command(help="Print Notoma version.")
def version():
    click.echo(f"Notoma {__version__}")


@runner.command(help="Convert Notion Blog to Markdown files.")
@click.option("--from", "-f", "notion_url", help="Notion blog URL")
@click.option(
    "--dest",
    "-d",
    required=True,
    default="posts",
    type=click.Path(exists=True),
    help="Directory to put posts into.",
)
@click.option("--token_v2", "-t", help="Notion auth token from the cookie.")
@click.option("--verbose", is_flag=True, default=False, help="Verbose output.")
def convert(
    dest: str, verbose: bool = False, token_v2: str = None, notion_url: str = None
) -> None:
    config = Config(token_v2=token_v2, blog_url=notion_url)
    dest = Path(dest).absolute()
    client = notion_client(config.token_v2)
    blog = notion_blog_database(client, config.blog_url)

    # FIXME accessing get_rows() directly relies on notion api,
    # use a notoma wrapper instead
    with click.progressbar(blog.get_rows()) as bar:
        for page in bar:
            page2path(page, dest_dir=dest).write_text(page2md(page))


@runner.command()
def watch() -> None:
    """
    Watch for updates in the Notion Blog and update the Markdown posts
    """
    raise NotImplementedError("Watcher is not implemented yet.")


@runner.command()
def new() -> None:
    """
    Create a new Notion Blog
    """
    raise NotImplementedError("Creating a new blog is not implemented yet.")
