import os
from setuptools import setup, find_packages
from notoma.version import __version__ as notoma_version

ROOT = os.path.dirname(os.path.abspath(__file__))


def get_requirements(env):
    """
    Takes requirements from requirements.txt.
    The detault (produciton) environment is "base"
    """
    with open(f"requirements-{env}.txt") as fp:
        reqs = list()
        for lib in fp.read().split("\n"):
            if not lib.startswith("-") or lib.startswith("#"):
                reqs.append(lib.strip())
        return reqs


setup(
    name="notoma",
    version=notoma_version,
    author="Nate Gadzhibalaev",
    author_email="nate@respawn.io",
    url="https://github.com/xnutsive/notoma/",
    description="Write your blog articles in Notion. Notoma converts your Notion database pages to Markdown files.",
    long_description=open(os.path.join(ROOT, "README.md")).read(),
    long_description_content_type="text/markdown",
    zip_safe=False,
    python_requires=">=3.7",
    install_requires=get_requirements("base"),
    extras_require={"dev": get_requirements("dev")},
    license="Apache Software License 2.0",
    classifiers=[
        "Development Status :: 4 - Beta",
        "License :: OSI Approved :: Apache Software License",
        "Typing :: Typed",
    ],
    project_urls={
        "Documentation": "https://xnutsive.github.io/notoma/",
        "Source Code": "https://github.com/xnutsive/notoma/",
    },
    entry_points={
        "console_scripts": [
            "notoma-dev = notoma.dev:cli",
            "notoma = notoma.cli:runner",
        ]
    },
    include_package_data=True,
    packages=find_packages(),
)
