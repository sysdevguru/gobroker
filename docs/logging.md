# Logging

## Basic Architecture

```
  --------------
  |  gotrader  |  --                             ----------
  --------------   |                      -----  |   s3   |
                   |                      |      ----------
  --------------   |     -------------    |
  |  gobroker  |  -----  |  fluentd  |  ---
  --------------   |     -------------    |
                   |                      |      ----------
  --------------   |                      -----  |   ES   |
  |  algodash  |  --                             ----------
  --------------

```

## Application Logging

We use uber-go/zap for logging. It is already integrated with github.com/alpacahq/gopaca/log, so you can just use it. Right now, we don't have specific rules how to structure the key-value pairs of zap, but might better to have. All the application logs except debug level one will be sent to fluentd, and will be stored into the data storage.

## Error Logging / Management

### Purpose

To make it easy to investigate on production and development.

### Requirements
- Errors will be shown on docker logs, and also sent to fluentd.
- Errors are needed to be propagated.
  - error need to be wrapped with context if captured in the middle of the logic.
    - To make it easy to debug. To know what happened on lower layer while investigation.
  - `github.com/pkg/errors` is the standard or common way to handle it in go, so we choose it.
- Errors need to be recorded at upstream layers.
  - to record how errors are propagated.
  - not to record errors multiple times.
  - to make it easy to aggregate and rely error event to log manager.
- Errors need to be wrapped with application error on service layer.
    - For us, application error = `alpacahq/gobroker/gberrors`
    - To show human interpretable user friendly error messages, and also keep lower level log information.

## HTTP Logging

### Purpose

For production use, we want to do there stuffs in the future (or not).

- Complex rule based Rate Limit
- Performance optimization
- Monitoring suspicious activity.
- DoS attack protection/monitoring for account creation.
- API usage monitoring for user.

### Requirements

- HTTP Logs can be shown on stdout while development.
- Logs will be sent to fluentd.
- Data: We'll store information below as HTTP Log.
    - Path
    - Request Body
    - Remote Addr
    - Status Code
    - Client Name
    - Elapse Time
    - AccountID (nullable) ?
    - API KEY ID (nullable)
- We need to mask private information not to be logged.
    - password
    - secret key
    - social security number


## Log Delivery

### Production

We'll deliver logs to the storage on production, and we'll use fluentd for that purpose. Application will just emit log with specific namespace (ex - error, http, app, kpi) and does not need to know where to deliver.

### Development

While development, we also want to see these logs on local environment environment. So we will deliver logs to stdout to watch it from `docker logs` command.
