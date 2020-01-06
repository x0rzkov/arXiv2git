#!/bin/sh

set -x
set -e

case "$1" in

  'all')
  	python3 /opt/app/0.pull_repo_list.py
  	python3 /opt/app/1.fetch_README.py
  	python3 /opt/app/2.extract_arxiv_links.py
	;;

  'pull')
  	exec python3 /opt/app/0.pull_repo_list.py $@
	;;

  'fetch')
  	exec python3 /opt/app/1.fetch_README.py $@
	;;

  'extract')
  	exec python3 /opt/app/2.extract_arxiv_links.py $@
  	;;

  *)
  	exec /bin/true
	;;
esac

