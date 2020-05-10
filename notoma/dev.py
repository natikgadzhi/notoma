from pathlib import Path
from typing import Union
import click
from nbconvert.exporters import MarkdownExporter


NBS_PATH = (Path(__file__).absolute/"../nbs/")
DOCS_PATH = (Path(__file__).absolute/"../docs/")
TPL_FILE = (Path(__file__).absolute/"templates/extended-docs-md.tpl")


@click.command()
def docs():
    """Build documentation as a bunch of .md files in ./docs/"""

    raise NotImplementedError("This needs to be rewritten from fastai/nbdev")

    nbs = [f for f in NBS_PATH.glob("*.ipynb")]
    dest = DOCS_PATH

    for fname in nbs:
        print(f"Converting {fname}")
        convert_nb_to_md(fname, dest)


@click.group()
def cli():
    """Notoma dev tools."""
    pass


cli.add_command(docs)


def convert_nb_to_md(fname: Union[str, Path],
                     dest: Union[str, Path] = DOCS_PATH):
    pass

    metadata = dict()
    exporter = MarkdownExporter(template_file=TPL_FILE)

    # Setup exporter config

    markdown = exporter.from_filename(str(fname), resources=metadata)
    with open(str(dest/fname.stem + ".md"), "w") as f:
        f.write(markdown)
