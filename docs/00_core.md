---
test: value
---

<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
# default_exp core
```

</div>

</div>

# Notoma

> Write articles for any static gen blog, in Notion.
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#hide
from nbdev.showdoc import *
```

</div>

</div>

Notoma is a small tool that works with your Notion database and allows you to turn your Notion pages into a blog, or a website — with any static gen website engine and hosting platform you want.

Notoma is available as a stand alone CLI app, and a Python library. You can use it locally to prepare and preview your articles and commit them, or as a part of your remote build pipeline.

## Imports and dependencies
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#export
from typing import List, Dict, Union
from notion.client import NotionClient
from notion.collection import *
from notion.block import *
from pathlib import Path
from dotenv import load_dotenv, find_dotenv
```

</div>

</div>

## Config

Where will we store configuration, including sensitive data like auth tokens, and blog db name?
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#exports
class Config:
    """
    Wraps Notoma's settings in an object with easier access.
    Settings are loaded from `.env` file, and from the system environment.
    You can override them by providing kwargs when creating an instance of a config.

    `.env` keys are explicit and long, i.e. `NOTOMA_NOTION_TOKEN_V2`. `kwargs` key responsible for the token is just
    `token_v2`.
    """

    def __init__(self, **kwargs):
        """
        Loads config from a `.env` file or system environment.

        You can provide any kwargs you want and they would override environment config values.
        """
        load_dotenv(find_dotenv())
        self.__config = dict(token_v2 = os.environ.get('NOTOMA_NOTION_TOKEN_V2'),
                               blog_url = os.environ.get('NOTOMA_NOTION_BLOG_URL'))

        for k, v in kwargs.items():
            self.__config[k] = v

    @property
    def token_v2(self):
        return self.__config['token_v2']

    @property
    def blog_url(self):
        return self.__config['blog_url']

    def __getitem__(self, key):
        return self.__config[key]

    def __repr__(self):
        return '\n'.join(f'{k}: {v}' for k, v in self.__config.items())
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
show_doc(Config)
```

</div>
<div class="output_area" markdown="1">


<h2 id="Config" class="doc_header"><code>class</code> <code>Config</code><a href="" class="source_link" style="float:right">[source]</a></h2>

> <code>Config</code>(**\*\*`kwargs`**)

Wraps Notoma's settings in an object with easier access.
Settings are loaded from `.env` file, and from the system environment. 
You can override them by providing kwargs when creating an instance of a config.

`.env` keys are explicit and long, i.e. `NOTOMA_NOTION_TOKEN_V2`. `kwargs` key responsible for the token is just
`token_v2`.


</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
config = Config()
```

</div>

</div>

## Notion Client

Provides a thin wrapper around `notion-py` — an API wrapper library for the reverse-engineered Notion API. Pagify users that client to grab the blog contents from Notion.

The client utilizes authentication token from Notion's web version, `token_v2`, that you can grab from your browers cookies. 

> Notice: This token authorizes any code to do anything that you can do in the browser in Notion on your behalf, so don't share it publicly, and don't store it your code or git.
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#exports
def notion_client(token_v2:str) -> NotionClient:
    client = NotionClient(token_v2=config.token_v2)
    return client
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
show_doc(notion_client)
```

</div>
<div class="output_area" markdown="1">


<h4 id="notion_client" class="doc_header"><code>notion_client</code><a href="__main__.py#L2" class="source_link" style="float:right">[source]</a></h4>

> <code>notion_client</code>(**`token_v2`**:`str`)




</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
client = notion_client(config.token_v2)
```

</div>

</div>

## Blog Database

Build a helper that would take the client, search for the provided DB name, and return it, or error out if notfound.

Build a function that returns an iterator over all the pages in that database.
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#exports
def notion_blog_database(client: NotionClient, db_url:str) -> Collection:
    """Returns a Notion database, wraped into a `notion.Collection` for easy access to it's rows."""
    return client.get_collection_view(db_url).collection
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
show_doc(notion_blog_database)
```

</div>
<div class="output_area" markdown="1">


<h4 id="notion_blog_database" class="doc_header"><code>notion_blog_database</code><a href="__main__.py#L2" class="source_link" style="float:right">[source]</a></h4>

> <code>notion_blog_database</code>(**`client`**:`NotionClient`, **`db_url`**:`str`)

Returns a Notion database, wraped into a `notion.Collection` for easy access to it's rows.


</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
blog = notion_blog_database(client, config.blog_url)
```

</div>

</div>

## Page Structure

Build a class or a function that takes the notion page object (from the iterator) and can return the metadata values, and iterate over it's contents.

`Page` is a record in the Blog database. Here's the format that Pagify expects: 

- Page title will be converted to the .md file name. File name will be formatted with dashes instead of spaces, and the page udpated at date will be uppended in YYYY-MM-DD format.

- Published: an optional boolean field. If present, Pagify will ignore pages where published: false. 

- Description: if the Page text starts with a word Desciption, then the whole first paragraph is considered description, and will be added to the markdown file front matter (metadata). 

- Publish at: a datetime field, if present, will be used as published at front matter key in the md file.


<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
page = blog.get_rows()[0]
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
page.title
```

</div>
<div class="output_area" markdown="1">




    'Notoma First Article'



</div>

</div>

## Converting Page to Markdown

Build a quick conversion of a Notion page to a .md file
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#export
def block2md(block:Block, counter:int = 1) -> str:
    """Transforms a Notion Block into a Markdown string."""

    if isinstance(block, TextBlock):
        return block.title

    elif isinstance(block, HeaderBlock):
        return f"# {block.title}"

    elif isinstance(block, SubheaderBlock):
        return f"## {block.title}"

    elif isinstance(block, SubsubheaderBlock):
        return f"### {block.title}"

    elif isinstance(block, QuoteBlock):
        return f"> {block.title}"

    elif isinstance(block, BulletedListBlock):
        return f"- {block.title}"

    elif isinstance(block, NumberedListBlock):
        return f"{counter}. {block.title}"

    elif isinstance(block, CodeBlock):
        return f"""
```{block.language}
{block.title}
```
"""

    elif isinstance(block, CalloutBlock):
        return f"> {block.icon} {block.title}"

    elif isinstance(block, DividerBlock):
        return "\n"
    else:
        return ""
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#export
def page2md(page:PageBlock) -> str:
    """Translates a Notion Page (`PageBlock`) into a Markdown string."""
    blocks = list()

    # Numbered lists iterator
    counter = 1

    for block in page.children:
        blocks.append(block2md(block, counter))

        if isinstance(block, NumberedListBlock):
            counter += 1
        else:
            counter = 1

    return page_front_matter(page) + "\n".join(blocks)
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#export
def page2path(page:PageBlock, dest_dir:Path=Path(".")) -> Path:
    """Build a .md file path in `dest_dir` based on a Notion page metadata."""
    return dest_dir/Path("-".join(page.title.lower().replace(".", "").split(" "))+ ".md")
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#export
def page_front_matter(page: PageBlock) -> str:
    """Builds a page front matter in a yaml-like format."""
    internals = ['published', 'title']
    renderables = { k:v for k,v in page.get_all_properties().items() if k not in internals }

    return f"""
---
{yaml.dump(renderables)}
---\n
"""
```

</div>

</div>

## Converting multiple pages

<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python
#exports
def notion2md(token_v2:str, database_url:str, dest:Union[str, Path]) -> None:
    """
    Grab Notion Blog database using auth token `token_v2`,
    convert posts in database `database_url` to Markdown, and save them to `dest`.
    """

    client = notion_client(token_v2)

    database = notion_blog_database(client, database_url)

    for post in database.get_rows():
        page2path(page, dest_dir=dest).write_text(page2md(page))
```

</div>

</div>
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python

```

</div>

</div>
