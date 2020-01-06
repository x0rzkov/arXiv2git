#!/bin/sh

set -x
set -e

case "$1" in

  'pull')
  	exec /opt/app/0.pull_repo_list.py $@
	;;

  'fetch')
  	exec /usr/bin/1.fetch_README.py $@
	;;

  'extract')
  	exec /usr/bin/2.extract_arxiv_links.py $@
  	;;

  *)
  	exec /bin/true
	;;
esac

