INSTALLDEST=/usr/local/bin
GO=/usr/local/go/bin/go

TESTFILE=_skull_supporto_meshmixer.photon
build:
	${GO} build -o photos main.go

install: uninstall build 
	ln -s `pwd`/photos ${INSTALLDEST}/photos

uninstall:
	-rm ${INSTALLDEST}/photos

.PHONY: test
test:
	cd test && bash upload_download_test.sh ${TESTFILE}
