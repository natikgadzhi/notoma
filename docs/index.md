---
layout: default

nav_order: 1

title: Install & Usage
---



# Notoma

> Write articles for your static gen blog in Notion.



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

### Authenticating in Notion

Since Notion doesn't yet (May 2020) have a public API, Notoma requires your Notion cookie auth token that you can get from your browser:
{: sp-4}



### Setting up the Blog database

Run `notoma create --name Blog` to create a new Blog database in the root of your Notion account.  

### Converting your Notion articles
