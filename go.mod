module teamvite

go 1.17

require internal/session v1.0.0

replace internal/session => ./internal/session

require (
	github.com/jmoiron/sqlx v1.3.4
	github.com/julienschmidt/httprouter v1.3.0
	github.com/mattn/go-sqlite3 v1.14.8
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
)

require (
	github.com/dchest/uniuri v0.0.0-20160212164326-8902c56451e9 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/lib/pq v1.10.3 // indirect
)
