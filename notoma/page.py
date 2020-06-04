from pathlib import Path
from notion.collection import NotionDate
from notion.block import PageBlock
from .config import Config


def page_path(page: PageBlock, dest_dir: Path = Path(".")) -> Path:
    "Build a .md file path in `dest_dir` based on a Notion page metadata."
    fname = "-".join(page.title.lower().replace(".", "").split(" ")) + ".md"
    return dest_dir / fname


def front_matter(page: PageBlock, config: Config = Config()) -> str:
    "Builds and returns a page front matter in a yaml-like format."
    all_props = page.get_all_properties()
    if "layout" not in all_props:
        all_props["layout"] = config.default_layout
    renderables = {k: v for k, v in all_props.items() if v != ""}
    return __sanitize_front_matter(renderables)


def __sanitize_front_matter(items: dict) -> dict:
    "Sanitizes and returns front matter items."
    for k, v in items.items():
        if type(v) not in [str, list]:
            if isinstance(v, NotionDate):
                items[k] = v.start
            if isinstance(v, bool):
                items[k] = str(v).lower()
    return items
