# About

Crosscraft - word guessing game implemented with golang.

# Config

Service is configured using environemnt vars:
* CROSSCRAFT_RECAPTCHA_KEY - reCaptcha key which is used to generate reCaptcha challenge on page
* CROSSCRAFT_RECAPTCHA_SECRET - reCaptcha secret for verifying the challenge on backend

# TODOs

* Output recaptcha challenge errors on welcome page.
* Access logs with method execution time.
* Add query id to logging.
* Add unit tests. 