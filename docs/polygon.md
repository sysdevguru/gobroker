# Polygon Integration

## Basic Architecture

```
-----------------   req data    ---------------
|   user algo   | ------------> |   polygon   |
-----------------  w/ key ID    ---------------
        |                              |
        |                              |    verify key ID is valid
        |                              |    and status is active
        |                              |
        |  generate api key     --------------
        --------------------->  |  gobroker  |
             ID & secret        --------------
```

## Current Implementation

In order to make things as seamless as possible for our users, they are able to use the same API key ID to request data from polygon that is used to interact with our trading API (plus the secret key). In order to make this possible, the above diagrammed architecture was implemented. A user generates their API key ID and secret through Alpaca's dashboard. The user then uses this key ID as their token for querying Polygon's API. Polygon then verifies the token with GoBroker's internal Polygon specific API (/auth endpoint) by making sure the key ID is in our DB and its status is ACTIVE. The Polygon specific API is secured using a secret key that we have provided to Polygon, as well as by source IP, where we have whitelisted their production IP block.

In order to verify the identities of the users who have been querying the data for reporting purposes, Polygon posts a list of API keys to our /keys endpoint. We then return the required data for each user who has viewed the data using those keys.

## Risks

Currently the main risk is the usage of only the API key ID to verify requests with Polygon. Theoretically, if a user's API key ID is stolen, another malicious user could request their data and be using it as much as they want until the user generates a new key-pair.