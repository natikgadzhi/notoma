# Notoma

> Write articles for your static gen blog in Notion.



## Work on progress! 

This is a super early version of Notoma — and this document is half true and half fiction.


## Install

Notoma is available via Pip and Homebrew: 

```bash
# Installing with pip, use this if you plan using Notoma as a python library.
pip install notoma
```

```bash
# Installing with brew
# Use this if you just want the CLI tool
brew install cask/xnutsive/notoma
```

## Basic Usage

Notoma provides commands to: 
- Convert contents of your Notion Blog database to a bunch of Markdown files.
- Watch Notion Blog database for updates and regenerate Markdown files on any updates.
- Create a new Notion database for your Blog with all required fields.

Here's the basic usage example: 

```bash
notoma convert --dest ./posts/
```

This example assumes that you have a `.env` config file with authentication and blog url parameters in it, this guide covers those later.

#### Authenticating in Notion

Since Notion doesn't yet (May 2020) have a public API, Notoma requires your Notion cookie auth token that you can get from your browser.

You can provide `token_v2` option to every command line call, or store it in your environment, or `.env` file.

#### Converting your Notion articles

`notoma convert --help`
<div class="codecell" markdown="1">
<div class="input_area" markdown="1">

```python

```

</div>

</div>
