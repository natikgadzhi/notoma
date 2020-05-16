from pathlib import Path
from typing import Union
import click
from nbconvert.exporters import MarkdownExporter
from nbconvert.preprocessors import RegexRemovePreprocessor
from nbdev.export import read_nb


NBS_PATH = Path(__file__).parent / "../notebooks/"
DOCS_PATH = Path(__file__).parent / "../docs/"
TPL_FILE = str(Path(__file__).parent / "templates/extended-docs-md.tpl")


@click.group()
def cli():
    pass


@cli.command()
def docs():
    """
    Build documentation as a bunch of .md files in ./docs/, and the README.md
    """
    nbs = [f for f in NBS_PATH.glob("*.ipynb")]

    for fname in nbs:
        fname = Path(fname).absolute()
        dest = Path(f"{DOCS_PATH/fname.stem}.md").absolute()
        print(f"Converting {fname} to {dest}")
        _convert_nb_to_md(fname, dest)


def _get_metadata(notebook: list) -> dict:
    "Find the cell with title and summary in `cells`."

    if not notebook["cells"]:
        raise ValueError("Expected the input to be NotebookCell-like list")

    markdown_cells = [
        cell["source"]
        for cell in notebook["cells"]
        if cell["cell_type"] == "markdown"
    ]

    meta = dict(layout="default")

    for cell in markdown_cells:
        if cell.startswith("%METADATA%"):
            for line in cell.split("\n")[1:]:
                k, v, *rest = [part.strip().lower() for part in line.slit(":")]
                meta[k] = v
    return meta


def _convert_nb_to_md(
    fname: Union[str, Path], dest: Union[str, Path] = DOCS_PATH
) -> None:
    notebook = read_nb(str(fname))
    metadata = _get_metadata(notebook)
    exporter = _build_exporter()

    prep = RegexRemovePreprocessor()
    prep.patterns = [r"![\s\S]", "$%METADATA%", "^#hide"]
    notebook, _ = prep.preprocess(notebook, {})

    converted = exporter.from_notebook_node(
        notebook, resources={"meta": metadata}
    )
    with open(str(dest), "w") as f:
        f.write(converted[0])


def _build_exporter() -> MarkdownExporter:
    exporter = MarkdownExporter(template_file=TPL_FILE)
    exporter.exclude_input_prompt = True
    exporter.exclude_output_prompt = True
    return exporter


def _make_readme(fname: Union[str, Path]):
    _convert_nb_to_md(fname, fname.parent.parent / "README.md")
