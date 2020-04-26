from setuptools import setup, find_packages
import os

ROOT = os.path.dirname(os.path.abspath(__file__))
VERSION = "0.0.1.dev0"


def get_requirements(env):
    with open(f"requirements-{env}.txt") as fp:
        reqs = list()
        for lib in fp.read().split("\n"):
            if not lib.startswith("-") or lib.startswith("#"):
                reqs.append(lib.strip())
        return reqs

install_requires = get_requirements("base")
dev_requires = get_requirements("dev")

setup(
    name="Notoma",
    version=VERSION,
    author="Nate Gadzhibalaev",
    author_email="nate@respawn.io",
    url="https://github.com/xnutsive/notoma/",
    description="Notion to markdown",
    long_description=open(os.path.join(ROOT, "README.md")).read(),
    long_description_content_type="text/markdown",
    zip_safe=False,
    install_requires=install_requires,
    extras_require={"dev": dev_requires},
    license="Apache Software License 2.0",
    entry_points = { 'console_scripts': 
        ["notoma-dev = notoma.scripts:cli"] },
    include_package_data=True,
    packages=find_packages()
)
