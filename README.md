# uptime monitor app 

simple uptime monitor that checks sites for successful get code 200
and sends an email if it it's unsuccessful.

2024-12-12 - 1.0.4 
- added response body check for apache/nginx placeholder pages. body check implemented, header check requires app-monitor coordination

2024-12-11 - 1.0.3
- Added prometheus printout on 8080 to make public-facing panels on grafana that keep track of uptime 
- added semver and cicd 
