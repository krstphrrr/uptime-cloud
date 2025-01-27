# uptime monitor app 

simple uptime monitor that checks sites for successful get code 200
and sends an email if it it's unsuccessful.

2025-01-24 - 1.0.9
- sendalert function now is aware of polled application status

2025-01-24 - 1.0.8
- leveraging alert state

2025-01-24 - 1.0.7
- added success threshold

2025-01-24 - 1.0.6
- testing separate config json
- iterate over custodians to maintain custodian privacy. 


2025-01-24 - 1.0.5
- added debounce time + failure threshold
- made poll function aware of error codes

2024-12-12 - 1.0.4 
- added response body check for apache/nginx placeholder pages. body check implemented, header check requires app-monitor coordination

2024-12-11 - 1.0.3
- Added prometheus printout on 8080 to make public-facing panels on grafana that keep track of uptime 
- added semver and cicd 
