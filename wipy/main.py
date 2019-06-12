#!/usr/bin/env python
#
# Copyright (c) 2019, Pycom Limited.
#
# This software is licensed under the GNU GPL version 3 or any
# later version, with permitted additional terms. For more information
# see the Pycom Licence v1.0 document supplied with this file, or
# available at https://www.pycom.io/opensource/licensing
#

# See https://docs.pycom.io for more information regarding library specifics

import time
import machine
import ubinascii
import sys

from network import WLAN
from network import Bluetooth
import usocket as socket

from pysense import Pysense
from LIS2HH12 import LIS2HH12
from SI7006A20 import SI7006A20
from LTR329ALS01 import LTR329ALS01
from MPL3115A2 import MPL3115A2,ALTITUDE,PRESSURE
from secrets import WIFISSID, WIFIPASS
wdt = machine.WDT(timeout=300000)

py = Pysense()
mp = MPL3115A2(py,mode=ALTITUDE) # Returns height in meters. Mode may also be set to PRESSURE, returning a value in Pascals
si = SI7006A20(py)
lt = LTR329ALS01(py)
li = LIS2HH12(py)

bt = Bluetooth()
bt.set_advertisement(name="pycom")
bt.advertise(True)

while True:
    wdt.feed()
    bt.start_scan(10)
    while bt.isscanning():
        time.sleep_ms(100)
    advs = bt.get_advertisements()
    devices = set()
    devicetypes = dict()
    for adv in advs:
        if adv.mac in devices:
            continue
        devices.add(adv.mac)
        print("MAC: "+str(ubinascii.hexlify(adv.mac)))
        print("RSSI: "+str(adv.rssi))
        print("connecting to "+str(ubinascii.hexlify(adv.mac)))
        try:
            conn = bt.connect(adv.mac)
            if conn:
                try:
                    print("connected to "+str(ubinascii.hexlify(adv.mac)))
                    services = conn.services()
                    for service in services:
                        if type(service.uuid()) == bytes:
                            print("Srv: "+str(ubinascii.hexlify(service.uuid())))
                        else:
                            print('Srv: %x' % service.uuid())
                        if service.uuid() == 6154: # 0x180A - Device Information
                            for c in service.characteristics():
                                if c.uuid() == 10788: # 0x2A24 - Model Number String
                                    model = c.read().decode("utf-8")
                                    print("model: "+model)
                                    t = "other"
                                    if model.startswith("Mac"):
                                        t = "Mac"
                                    elif model.startswith("iPhone"):
                                        t = "iPhone"
                                    if t not in devicetypes:
                                        devicetypes[t] = 1
                                    else:
                                        devicetypes[t] += 1
                except OSError as e:
                    print(e)
                except:
                    e = sys.exc_info()[0]
                    print(e)
                finally:
                    print("disconnecting from "+str(ubinascii.hexlify(adv.mac)))
                    conn.disconnect()
        except OSError as e:
            print(e)
        except:
            e = sys.exc_info()[0]
            print(e)
    temp = si.temperature()
    hum = si.humidity()
    lum = lt.light()
    msg = 'alive,room=pysense value=1\ntemp,room=pysense value=%f\nhumidity,room=pysense value=%f\nlum,room=pysense value=%f\ndevices,room=pysense value=%f\n'%(temp, hum, lum[0]+lum[1], len(devices))
    for (k, v) in devicetypes.items():
        msg += "devicetype,room=pysense,type=%s value=%d\n"%(k, v)
    print("Temperature: " + str(temp)+ " deg C and Relative Humidity: " + str(hum) + " %RH")
    print("Light (channel Blue lux, channel Red lux): " + str(lum))

    print("init WLAN...");
    wlan = WLAN(mode=WLAN.STA)
    print("connecting to wifi...");
    wlan.ifconfig(config='dhcp')
    wlan.connect(WIFISSID, auth=(WLAN.WPA2, WIFIPASS), timeout=5000)
    while not wlan.isconnected():
         machine.idle()
    print("Connected as "+str(wlan.ifconfig()))

    req = "POST /influxdb/write?db=demo HTTP/1.0\nHost: 35.198.66.194:2015\nConnection: close\n"
    req += "Content-Type: application/x-www-form-urlencoded\nContent-Length: %d\n\n"%len(msg)
    req += msg
    print(req)

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    ai = socket.getaddrinfo("35.198.66.194", 2015)
    addr = ai[0][4]
    s.connect(addr)
    s.sendall(req)
    status = s.recv(4096)
    s.close()
    print(str(status))
    wlan.disconnect()
    time.sleep_ms(60000)
