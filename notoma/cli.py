import os
from pathlib import Path

import click
import requests

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
from .logging import get_logger, toggle_debug, LOG_FILE_HANDLER, LOG_FMT
from logging import INFO, DEBUG, ERROR

logger = get_logger(INFO, LOG_FILE_HANDLER, LOG_FMT)

"""
`cli` Module only has thin wrappers around Notoma Python API
that invokes the API with provided configuration.

Run `notoma` for the help article on how to use the CLI.
"""


# The CLI runner
#
@click.group(
    help="""
    Build your staticg gen blog with Notion.
    Notoma converts Notion database of blog posts into a directory of .md files.
    """
)
@click.option("--debug", is_flag=True, default=False, help="Enable debug output.")
def runner(debug: bool = False) -> None:
    logger.info("Notoma CLI invoked.")
    toggle_debug(logger, debug)
    pass


@runner.command(help="Print Notoma version.")
def version():
    __echo_and_log(f"Notoma {__version__}")


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
def convert(
    dest: str, drafts: str, token_v2: str = None, notion_url: str = None,
) -> None:
    config = Config(token_v2=token_v2, blog_url=notion_url)
    __validate_config(config)

    dest = Path(dest).absolute()
    client = notion_client(config.token_v2)
    blog = notion_blog_database(client, config.blog_url)

    __echo_and_log(f"Processing articles from Notion: {blog.parent.title}")

    try:
        __convert_pages(published_pages(blog), dest, config)
        if drafts:
            __convert_pages(draft_pages(blog), Path(drafts).absolute(), config)
            draft_pages(blog)

    except requests.exceptions.HTTPError as e:
        __echo_and_log(e, ERROR)


@runner.command()
def watch() -> None:
    """
    Watch for updates in the Notion Blog and update the Markdown posts
    """
    logger.info("Invoking Watch command.")

    error = NotImplementedError("Watcher is not implemented yet.")
    logger.error("Watch is not implemented.", error)
    raise error


@runner.command()
def new() -> None:
    """
    Create a new Notion Blog
    """
    logger.info("Invoking Watch command.")

    error = NotImplementedError("Creating a new blog is not implemented yet.")
    logger.error("Watch is not implemented.", error)
    raise error


def __convert_pages(pages: list, dest_dir: Path, config: Config) -> None:
    "Convert a bunch of pages with a nice progress bar."

    __echo_and_log(f"{len(pages)} pages to process.")

    with click.progressbar(pages) as bar:
        for page in bar:
            path = page_path(page, dest_dir=dest_dir)
            page_markdown = page_to_markdown(page, config=config)
            path.write_text(page_markdown)
            logger.info(f"Processed page {page.title} and saved to {path}.")

    __echo_and_log(f"Processed {len(pages)} pages.")


def __validate_config(config: Config) -> None:
    """
    Validates the provided options and prints errors to stdout,
    then aborts if there are any errors.
    """
    errors = list()
    if config.token_v2 is None:
        errors.append(
            "Error: Authentication token `token_v2` required. Try --token option."
        )

    if config.blog_url is None:
        errors.append("Error: Notion Blog URL is required. Try --from option.")

    if len(errors) > 0:
        for e in errors:
            __echo_and_log(e, ERROR)
        raise click.Abort()


def __echo_and_log(message: str, loglevel=INFO) -> None:
    "Echo the message, and add it to the log with loglevel."
    if logger.level <= loglevel:
        click.echo(message)
    logger.log(loglevel, message)
