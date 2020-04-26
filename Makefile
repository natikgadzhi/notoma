SRC = $(wildcard ./*.ipynb)

all: notoma docs

notoma: $(SRC)
	pipenv run notoma-dev build
	touch notoma

docs_serve: docs
	cd docs && bundle exec jekyll serve

docs: $(SRC)
	pipenv run notoma-dev build-docs
	touch docs

test:
	nbdev_test_nbs

release: pypi
	nbdev_bump_version

pypi: dist
	twine upload --repository pypi dist/*

dist: clean
	python setup.py sdist bdist_wheel

clean:
	rm -rf dist
