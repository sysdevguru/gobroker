### Flow
Branch structure is as follows:

```
[ production release ]        >------------     v1.x.x     ------------<
[ production ready ]          >------------     master     ------------<
[ stable (stg verified) ]     >------------     develop    ------------<
[ unstable (stg unverified)]  >------------  candidate/v2  ------------<
```

*Note that versions 1 and 2 are placeholders for the current and next releases*

The general flow is that any new features will initially go into a `candidate/v2` branch, where any potentially breaking changes and major features will be developed, and tested in staging before merging to `develop`. Once the new feature branch is merged to `develop`, it is assumed that all new features from `candidate/v2` have been thoroughly tested in the staging environment, and are stable. 

Once the new features have reached the stable state in `develop`, a production release may be cut by merging to `master`, and cutting a versioned tag. This will initiate deployment to the production k8s cluster.

If and when hotfixes are needed to be applied to a current production release, any fixes should be branched directly from `master`, and merged to both `master` and `develop` upon completion. It is intended that the new `develop` branch be tested in staging, and verified before merge to `master`, and subsequently deployed to production. Meanwhile, new feature development will continue in a `candidate/v3` branch, unimpeded by the hotfixes made to `develop`.

### Versioning
Versioning follows git flow as follows:

```
Major release (major new features):           v2.0.0
Minor release (minor new features + fixes):   v2.x.0
Hotfix release:                               v2.x.x
```
