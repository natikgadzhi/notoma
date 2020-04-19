# Notoma
> Use Notion to write articles for any blogging platform that works with Markdown. Pagify automatically exports your Notion pages into a bunch of `.md` files, available as a CLI tool, or a Python library.


```python

from notoma.core import *
from nbdev.showdoc import *
```

## Install

`pip install notoma`

## Basic Usage

### Getting the Notion auth token

Pagify uses [`notion-py`](https://github.com/jamalex/notion-py) to work with Notion's reverse engineered web api. It's not an official API, hence the unorthodox authentication techniques. 

`notion-py` wants an auth token, called `token_v2`, that you can get from a Cookie on notion.so.

### Using Pagify as a standalone app

You can install Pagify and use it as a standalone utility:

```bash
pip install notoma

pagify --token=TOKEN --db=URL --dest .

```

### Using from your python code

```python
show_doc(notion2md)
```


<h4 id="notion2md" class="doc_header"><code>notion2md</code><a href="https://github.com/xnutsive/pagify/tree/master/pagify/core.py#L73" class="source_link" style="float:right">[source]</a></h4>

> <code>notion2md</code>(**`token_v2`**:`str`, **`database_url`**:`str`, **`dest`**:`Union`\[`str`, `Path`\])

Grab Notion Blog database using auth token `token_v2`,
convert posts in database `database_url` to Markdown, and save them to `dest`.

