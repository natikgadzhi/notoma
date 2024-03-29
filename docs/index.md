---
layout: default
nav_order: 1
title: Introduction
---

<!--
THIS FILE IS AUTOGENERATED, DON'T EDIT.
File to edit instead: notebooks/index.ipynb
-->

# Notoma

Write articles for your static gen blog in Notion.



<a href="https://codeclimate.com/github/nategadzhi/notoma/maintainability"><img src="https://api.codeclimate.com/v1/badges/70943357e5d2c54c153a/maintainability" /></a>
<a href="https://pypi.org/project/notoma/"><img src="https://img.shields.io/pypi/v/notoma" alt="pypi" /></a>
![Linters](https://github.com/nategadzhi/notoma/workflows/Linters/badge.svg)

- [Documentation website](https://nategadzhi.github.io/notoma/)
    - [Using the CLI](https://nategadzhi.github.io/notoma/using-the-cli)
    - [Contributing](https://nategadzhi.github.io/notoma/contributing)
    - [Supported Markdown Tags](https://nategadzhi.github.io/notoma/supported-markdown-tags) 

---
## Install

Notoma is available via Pip or Homebrew: 

```bash
# Installing with pip, use this if you plan using Notoma as a python library.
pip install notoma
```

Installing with Homebrew on Mac OS.

```bash
brew install nategadzhi/notoma/notoma
```

---
## What can you do with Notoma
Notoma provides commands to: 
- Convert contents of your Notion Blog database to a bunch of Markdown files.
- *Coming soon*: Watch Notion Blog database for updates and regenerate Markdown files on any updates.
- *Coming soon*: Create a new Notion database for your Blog with all required fields.

Basic usage example: this command will convert only published posts from a Notion blog database to the `./posts/ directory`.

```bash
notoma convert --dest ./posts/
```

This example assumes that you have a `.env` config file with authentication and blog url parameters in it.

#### Authenticating in Notion

Notoma uses an internal Notion API, and that, unfortunately, requires you to provide an authentication token `token_v2` that you can find in your notion.so cookes.

You can provide `token_v2` option to every command line call, or store it in your environment, or [`.env` config file](.env.sample).

---
## Notion database structure
Notoma has very few expectations about how your Notion is structured. Here's a [public example database](https://www.notion.so/respawn/7b46cea379bd4d45b68860c2fa35a2d4?v=b4609f6aae0d4fc1adc65a73f72d0e21).

Notoma requires that your Notion blog database has the following **properties**:
- **Published**: whether the article is published, or is still a draft
- **Title**: Will be used to create a file name for that article's Markdown equivalent file. *Won't be used in the article itself.*

Notoma tries to parse other properties and add them as front matter into the resulting Markdown articles: 
- **Published at** will be used as publicataion date for the article, if present.
- **Categories** will be used as `categories` front matter key, so it's expected to be a **multiple choice** propery.

