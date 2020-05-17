import os
from dotenv import load_dotenv, find_dotenv


class Config:
    """
    Wraps Notoma's settings in an object.
    Settings are automatically loaded from ENV (and `.env` file), and you
    can override them with `kwargs`.

        - `token_v2`: str, Notion authentication token. Environment variable
            `NOTOMA_NOTION_TOKEN_V2`.
        - `blog_url`: str, Notion Blog URL. `NOTOMA_NOTION_BLOG_URL`.
    """

    def __init__(self, **kwargs):
        """
        Loads config from a `.env` file or system environment.

        You can provide any kwargs you want and they would override
         environment config values.
        """
        load_dotenv(find_dotenv())
        self.__config = {
            "token_v2": os.environ.get("NOTOMA_NOTION_TOKEN_V2"),
            "blog_url": os.environ.get("NOTOMA_NOTION_BLOG_URL"),
        }

        for key, value in kwargs.items():
            if value is not None:
                self.__config[key] = value

    @property
    def token_v2(self):
        return self.__config["token_v2"]

    @property
    def blog_url(self):
        return self.__config["blog_url"]

    def __getitem__(self, key):
        return self.__config[key]

    def __repr__(self):
        return "\n".join(f"{k}: {v}" for k, v in self.__config.items())
