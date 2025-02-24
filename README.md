# sqlite-server
# Purpose:
This repo is contains the backend implementation of a SQLite GUI aimed at allowing a server-like experience
while manipulating SQLite databases on the web.

The primary improvement in this SQLite solution is the server's ability to:
1. DB per user (in addition to file permissions, allows a completely containerized storing of user data)
2. Allowing all new dbs created by a user to be visable and accessible through queries by attaching them prior 
to running queries. Example: `SELECT d.model, e.severity FROM  car_db.car_details as d JOIN traffic_accidents_db.events as e
ON d.model = e.offender_model WHERE e.severity IN ('critical', 'fatal');`
