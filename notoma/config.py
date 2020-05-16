import os
from dotenv import load_dotenv, find_dotenv


class Config:
    """
    Wraps Notoma's settings in an object with easier access.
    Settings are loaded from `.env` file, and from the system environment.
    You can override them by providing kwargs when creating an
     instance of a config.

    `.env` keys are explicit and long, i.e. `NOTOMA_NOTION_TOKEN_V2`.
    `kwargs` key responsible for the token is just `token_v2`.
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

        for k, v in kwargs.items():
            self.__config[k] = v

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
