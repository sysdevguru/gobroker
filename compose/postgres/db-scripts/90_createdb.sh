#!/bin/bash

echo "creating the db for gobroker"

cd /docker-entrypoint-initdb.d

psql <<- EOSQL
  CREATE DATABASE gobroker;
  \c gobroker;
  CREATE EXTENSION pg_stat_statements;
  CREATE EXTENSION sslinfo;
  CREATE DATABASE gotrader TEMPLATE gobroker;
  CREATE DATABASE gotrader_sim TEMPLATE gotrader;
  CREATE DATABASE gbtest TEMPLATE gobroker;
  CREATE DATABASE papertrader TEMPLATE gobroker;
  CREATE DATABASE "TRAFIX" TEMPLATE gotrader;
EOSQL

echo "done"
