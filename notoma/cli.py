import os
from pathlib import Path
import click

from .config import Config
from .core import (
    notion_client,
    notion_blog_database,
    page_to_markdown,
    page_path,
    published_pages,
    draft_pages,
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
@click.option(
    "--drafts",
    default=None,
    type=click.Path(exists=True),
    help="Directory for draft posts. Drafts won't be imported if this is left blank.",
)
@click.option("--token_v2", "-t", help="Notion auth token from the cookie.")
@click.option("--verbose", is_flag=True, default=False, help="Verbose output.")
@click.option("--debug", is_flag=True, default=False, help="Enable debug output.")
def convert(
    dest: str,
    drafts: str,
    debug: bool = False,
    verbose: bool = False,
    token_v2: str = None,
    notion_url: str = None,
) -> None:
    config = Config(token_v2=token_v2, blog_url=notion_url)
    dest = Path(dest).absolute()
    client = notion_client(config.token_v2)
    blog = notion_blog_database(client, config.blog_url)

    published = published_pages(blog)
    with click.progressbar(published) as bar:
        for page in bar:
            page_path(page, dest_dir=dest).write_text(page_to_markdown(page))

    if verbose:
        click.echo(f"Processed {len(published)} published pages.")

    if drafts:
        drafts = Path(drafts).absolute()
        drafted_pages = draft_pages(blog)
        with click.progressbar(drafted_pages) as bar:
            for page in bar:
                page_path(page, dest_dir=drafts).write_text(page_to_markdown(page))
        if verbose:
            click.echo(f"Processed {len(drafted_pages)} drafts.")


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
