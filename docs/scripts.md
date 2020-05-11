---
test: value
---

<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#default_exp scripts
```

</div>

</div>

# Dev tools

- Summary: Handy dev tools for Notoma.
- layout: default
- title: Dev Tools

Notoma is built using `nbdev` — meaning I write code and documentation as Jupyter notebooks and then extract the code I need into the library, package and push it to pypi, generate the docs automatically, and use the notebooks as tests. 

The problem though is that `nbdev` is very opinionated about how specifically I have to run tests and how what Jekyll theme my docs website has to use. Notoma provides a few small CLI tools to work around some of the assumtions `nbdev` makes and make the workflow a bit more generic.
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#exports

import click
from pathlib import Path
from nbdev.export import notebook2script
from nbdev.export2html import convert_md
from nbdev.imports import parallel, Config

```

</div>

</div>

This module provides a CLI tool `notoma-dev` that will be available when you install notoma using pip.
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#export
@click.command()
def build_docs():
    """Build documentation as a bunch of .md files in ./docs/"""
    
    nbs = [f for f in Config().nbs_path.glob('*.ipynb') if not f.name.startswith('_')]
    #docs = [ Path(f"{Config().doc_path}/{nb}").stem + '.md' for nb in nbs]
    
    dest = Config().doc_path
    
    for fname in nbs:
        print(f"Converting {fname}")
        convert_md(fname, dest)


    # notebook2html(cls=MarkdownExporter, template_file='markdown.tpl', force_all=True)
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#export
@click.command()
def build():
    """Rebuild python lib from notebooks using nbdev."""
    notebook2script()
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#export
@click.group()
def cli():
    """Notoma dev tools."""
    pass
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#export
cli.add_command(build_docs)
cli.add_command(build)
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
notebook2script()
```

</div>
<div class="output_area" markdown="1">

    Converted 00_core.ipynb.
    Converted 01_dev_scripts.ipynb.
    Converted index.ipynb.


</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
convert_md??
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python

```

</div>

</div>
