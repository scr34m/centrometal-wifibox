Centrometal WifiBox Command Server
---

The goal of this project is to read all data that would normally be exported to the original server and store it localy.
This program is handling my Pellet Set 25KW and so Home Assistant can use walues with out internet connection.

The WifiBox connects to this server (which acts as an MQTT broker), and the parsed information is then sent back to the local MQTT server.

You can either forward the port on your router or change the DNS resolution of portal.centrometal.hr to the local server’s IP address.

Missing functions:
- programming table: 327 table 1, 328 table 2, 329 table 3

Two topic is unique:
- `centrometal/command` which is accepts `CMD ON`, `CMD OFF`, `REFRESH`, `RSTAT`. The `PRD` and `PWR` commands value is seprated with a colon ex.: `PRD 327:ALL`
- `centrometal/_e_w_status` is holding the last error or warning code until its acknowledged on the boiler
