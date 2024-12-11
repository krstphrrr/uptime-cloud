# uptime monitor app 

simple uptime monitor that checks sites for successful get code 200
and sends an email if it it's unsuccessful.

- Added prometheus printout on 8080 to make public-facing panels on grafana that keep track of uptime 
- need to add header check 
- timeout after unsuccessful check
- semver and cicd