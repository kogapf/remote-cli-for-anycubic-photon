INSTALLDEST=/usr/local/bin
GO=/usr/local/go/bin/go
build:
	${GO} build -o photos main.go

install: uninstall build 
	ln -s `pwd`/photos ${INSTALLDEST}/photos

uninstall:
	-rm ${INSTALLDEST}/photos

