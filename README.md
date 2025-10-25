Centrometal WifiBox Command Server
---

The goal of this project is to read all data that would normally be exported to the original server and store it localy.

he WifiBox connects to this server (which acts as an MQTT broker), and the parsed information is then sent back to the local MQTT server.

You can either forward the port on your router or change the DNS resolution of portal.centrometal.hr to the local server’s IP address.

Missing functions:
- start / stop
- refresh
- programming table
