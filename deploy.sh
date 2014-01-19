DEST=../openshift/cmo

GOPATH=`pwd` go build -o bin/cmo app && strip -s bin/cmo

rsync bin/cmo $DEST/bin/cmo
rsync --recursive --delete static/ $DEST/static/
rsync --recursive --delete src/app/tmpl/ $DEST/src/app/tmpl/
rsync --recursive --delete src/app/langs/ $DEST/src/app/langs/
cd $DEST
git add .
git ci -m "new version"
git push
