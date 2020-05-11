from pathlib import Path
from typing import Union
import click
from nbconvert.exporters import MarkdownExporter
from nbdev.export2html import execute_nb


NBS_PATH = (Path(__file__).parent/"../notebooks/")
DOCS_PATH = (Path(__file__).parent/"../docs/")
TPL_FILE = str(Path(__file__).parent/"templates/extended-docs-md.tpl")


@click.group()
def cli(): pass


@cli.command()
def docs():
    """Build documentation as a bunch of .md files in ./docs/"""
    nbs = [f for f in NBS_PATH.glob("*.ipynb")]

    for fname in nbs:
        fname = Path(fname)
        dest = Path(f"{DOCS_PATH/fname.stem}.md")
        print(f"Converting {fname} to {dest}")
        _convert_nb_to_md(fname, dest)


def _convert_nb_to_md(fname: Union[str, Path],
                      dest: Union[str, Path] = DOCS_PATH) -> None:

    # TODO: Fetch the metadata from the notebook
    metadata = dict(test="value")
    exporter = _build_exporter()

    # TODO: Preprocess cells:
    #       - Execute the notebook to make sure it works correctly
    #       - Drop cells with metadata from the exported docs
    #       - See if I can get show_doc to work in my nbs
    #       - Preprocess any links / backtricks with links to source
    converted = exporter.from_filename(str(fname), resources={"meta": metadata})

    with open(str(dest), "w") as f:
        f.write(converted[0])


def _build_exporter() -> MarkdownExporter:
    exporter = MarkdownExporter(template_file=TPL_FILE)
    exporter.exclude_input_prompt = True
    exporter.exclude_output_prompt = True
    return exporter


def _make_readme(fname: Union[str, Path]):
    _convert_nb_to_md(fname, fname.parent.parent/"README.md")
