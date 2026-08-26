# Security policy

Report vulnerabilities privately to the repository maintainer before public
disclosure. Do not include pairing tokens, private addresses, or raw TV logs in
issues. Pairing tokens are credentials and must never be committed.

The plugin talks only to the configured television on TCP 8001/8002 and sends
Wake-on-LAN on UDP 9. SSDP discovery sends to the standard multicast address
and does not create a persistent listener.
