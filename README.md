# About

Crosscraft - word guessing game implemented with golang.

# Config

Service is configured using environemnt vars:
* CROSSCRAFT_RECAPTCHA_KEY - reCaptcha key which is used to generate reCaptcha challenge on page
* CROSSCRAFT_RECAPTCHA_SECRET - reCaptcha secret for verifying the challenge on backend
* CROSSCRAFT_DB_USER - database user name
* CROSSCRAFT_DB_PASSWORD - database password
* CROSSCRAFT_DB_NAME - database name
* CROSSCRAFT_DB_HOST - database hostname

# TODOs

* Output recaptcha challenge errors on welcome page.
* Add query id to logging.
* Add unit tests. 