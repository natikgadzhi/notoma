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
from . import __version__

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
    __validate_config(config)

    dest = Path(dest).absolute()
    client = notion_client(config.token_v2)
    blog = notion_blog_database(client, config.blog_url)

    if verbose:
        click.echo(f"Processing articles from Notion: {blog.parent.title}")

    __convert_pages(published_pages(blog), dest, config, verbose)

    if drafts:
        __convert_pages(draft_pages(blog), Path(drafts).absolute(), config, verbose)
        draft_pages(blog)


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


def __convert_pages(
    pages: list, dest_dir: Path, config: Config, verbose: bool = False
) -> None:
    "Convert a bunch of pages with a nice progress bar."

    if verbose:
        click.echo(f"{len(pages)} pages to process.")

    with click.progressbar(pages) as bar:
        for page in bar:
            page_path(page, dest_dir=dest_dir).write_text(
                page_to_markdown(page, config=config)
            )

    if verbose:
        click.echo(f"Processed {len(pages)} pages.")


def __validate_config(config: Config) -> None:
    """
    Validates the provided options and prints errors to stdout,
    then aborts if there are any errors.
    """
    errors = list()
    if config.token_v2 is None:
        errors.append(
            "Error: Authentication token `token_v2` is not provided. Try --token option."
        )

    if config.blog_url is None:
        errors.append(
            "Error: Notion Blog Database URL not provided. Try --from option."
        )

    if len(errors) > 0:
        for e in errors:
            click.echo(e)
        raise click.Abort()
