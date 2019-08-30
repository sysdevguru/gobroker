# Account Details Loader

## Usage

acctloader can be used to either create a new account from a yaml file, or update the details of an existing account with a given ID.

Create:

```
$ acctloader -e <email> -p <password> <yaml filepath>
$
```

Update:

```
$ acctloader -i <account id> <yaml filepath>
```

Or

```
$ acctloader  -i <account id> -d '
<yaml content>
'
```
