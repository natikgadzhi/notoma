all: notoma docs

clean-deps:
	rm -rf requirements-base.txt requirements-dev.txt

deps: clean-deps
	pipenv lock -r > requirements-base.txt
	pipenv lock -rd > requirements-dev.txt

install-deps: deps
	pipenv install
	pipenv install --dev

run-docs: docs
	cd docs && bundle exec jekyll serve


.PHONY: docs
docs:
	pipenv run notoma-dev docs
	touch docs

test:
	make test
	# pipenv run nbexec ./notebooks/*.ipynb

pypi: dist
	pipenv run twine upload --repository pypi dist/*

dist: clean
	python setup.py sdist bdist_wheel

clean:
	rm -rf dist
