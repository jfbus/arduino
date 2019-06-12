#include <SPI.h>
#include <WiFiNINA.h>

#include "Seeed_MCP9808.h"

#include "arduino_secrets.h"

char ssid[] = SECRET_SSID;        // your network SSID (name)
char pass[] = SECRET_PASS;    // your network password (use for WPA, or use as key for WEP)
int status = WL_IDLE_STATUS;     // the Wifi radio's status

WiFiClient client;

// temperature
MCP9808  sensor;
// sound
const int pinSound = A0;
const int pinLight = A1;


void setup() {
  //Initialize serial and wait for port to open:
  Serial.begin(9600);
  //while (!Serial) {
  //  ; // wait for serial port to connect. Needed for native USB port only
  //}
  Serial.println("Start setup");

  // check for the WiFi module:
  if (WiFi.status() == WL_NO_MODULE) {
    Serial.println("Communication with WiFi module failed!");
    // don't continue
    while (true);
  }
  String fv = WiFi.firmwareVersion();
  if (fv < "1.0.0") {
    Serial.println("Please upgrade the firmware");
  }

  // init bluetooth
  //pinMode(RxD, INPUT);    //UART pin for Bluetooth
  //pinMode(TxD, OUTPUT);   //UART pin for Bluetooth
  //if(setupBlueToothConnection() != 0) while(1);   //initialize Bluetooth
  //this block is waiting for connection was established.
  //expectBTResponse("OK+CONB", 20000);

  //sendBlueToothCommand("AT+DISC?", "?");//detect if the module exists
  //expectBTResponse("foo", 20000);

  // init temp
  if(sensor.init()) {
    Serial.println("sensor init failed!!");
  }
  Serial.println("sensor init!!");
}


int loopDelay = 60000;

void loop() {
  // get sound
  long sound = 0;
  for(int i=0; i<32; i++)
  {
      sound += analogRead(pinSound);
      delay(loopDelay/32);
  }
  sound >>= 5;
  // get Light
  int lum = analogRead(pinLight)/10;
  // get temperature
  float temp=0;
  sensor.get_temp(&temp);

  int start = millis();
  // connect or reconnect to wifi network
  status = WiFi.status();
  while (status != WL_CONNECTED) {
    Serial.print("Attempting to connect to WPA SSID: ");
    Serial.println(ssid);
    // Connect to WPA/WPA2 network:
    status = WiFi.begin(ssid, pass);

    // wait 10 seconds for connection:
    delay(10000);
  }
  Serial.print("You're connected to the network");
  printCurrentNet();
  printWifiData();

  String message = String("alive,room=arduino value=1\ntemp,room=arduino value=");
  message.concat(String(temp, 2));
  message.concat("\nnoise,room=arduino value=");
  message.concat(String(sound));
  message.concat("\nlum,room=arduino value=");
  message.concat(String(lum));
  message.concat("\n");

  Serial.println("connecting to server...");
  // if you get a connection, report back via serial:
  if (client.connect("35.198.66.194", 2015)) {
    Serial.println("sending request");
    Serial.println(message);

    // Make a HTTP request:
    client.println("POST /influxdb/write?db=demo HTTP/1.0");
    client.println("Host: 35.198.66.194:2015");
    client.println("Connection: close");
    client.print("Content-Length: ");
    client.println(message.length());
    client.println("Content-Type: application/x-www-form-urlencoded");
    client.println();
    client.print(message);

    String response = "";
    unsigned long time = 0;
    while (!response.startsWith("HTTP/") && time<1000) {
      while (client.available()) {
          char c = client.read();
          response += c;
      }
      delay(10);
      time += 10;
    }
    Serial.print(response);
  }
  WiFi.disconnect();
  loopDelay = 60000-(millis()-start);
  Serial.print("delay: ");
  Serial.println(loopDelay);
}

//
// WiFi functions
//
void printWifiData() {
  // print your board's IP address:
  IPAddress ip = WiFi.localIP();
  Serial.print("IP Address: ");
  Serial.println(ip);
  Serial.println(ip);

  // print your MAC address:
  byte mac[6];
  WiFi.macAddress(mac);
  Serial.print("MAC address: ");
  printMacAddress(mac);
}

void printCurrentNet() {
  // print the SSID of the network you're attached to:
  Serial.print("SSID: ");
  Serial.println(WiFi.SSID());

  // print the MAC address of the router you're attached to:
  byte bssid[6];
  WiFi.BSSID(bssid);
  Serial.print("BSSID: ");
  printMacAddress(bssid);

  // print the received signal strength:
  long rssi = WiFi.RSSI();
  Serial.print("signal strength (RSSI):");
  Serial.println(rssi);

  // print the encryption type:
  byte encryption = WiFi.encryptionType();
  Serial.print("Encryption Type:");
  Serial.println(encryption, HEX);
  Serial.println();
}

void printMacAddress(byte mac[]) {
  for (int i = 5; i >= 0; i--) {
    if (mac[i] < 16) {
      Serial.print("0");
    }
    Serial.print(mac[i], HEX);
    if (i > 0) {
      Serial.print(":");
    }
  }
  Serial.println();
}

//
// Bluetooth functions
//

//used for compare two string, return 0 if one equals to each other
int strcmp(char *a, char *b)
{
    unsigned int ptr = 0;
    while(a[ptr] != '\0')
    {
        if(a[ptr] != b[ptr]) return -1;
        ptr++;
    }
    return 0;
}

//configure the Bluetooth through AT commands
int setupBlueToothConnection()
{
    Serial.print("Setting up Bluetooth link\r\n");
    delay(3500);//wait for module restart

    //send command to module in different baud rate
    while(1)
    {
        delay(500);
        Serial1.begin(9600);
        delay(500);
        Serial.print("try 9600\r\n");
        if(sendBlueToothCommand("AT", "OK") == 0)
            break;
        delay(500);
        Serial1.begin(115200);
        delay(500);
        Serial.print("try 115200\r\n");
        if(sendBlueToothCommand("AT", "OK") == 0)
            break;
    }

    //we have to set the baud rate to 9600, since the soft serial is not stable at 115200
    sendBlueToothCommand("AT+DEFAULT", "OK+DEFAULT");//restore factory configurations
    sendBlueToothCommand("AT+NAMESeeedMaster", "OK+Set:SeeedMaster");//reset the module's baud rate
    sendBlueToothCommand("AT+ROLES", "OK+Set:S");//set to master mode
    sendBlueToothCommand("AT+FILT2", "OK+Set:2");//detect if the module exists
    sendBlueToothCommand("AT+NOTI1", "OK+Set:1");//detect if the module exists
    sendBlueToothCommand("AT+AUTH1", "OK+Set:1");//get firmware version
    sendBlueToothCommand("AT+CLEAR", "OK+CLEAR");//restart module to take effect
    delay(3500);//wait for module restart
    sendBlueToothCommand("AT", "OK");//detect if the module exists
    sendBlueToothCommand("AT+ADDR", "?");//detect if the module exists
    sendBlueToothCommand("AT+TEMP?", "?");//detect if the module exists
    Serial.print("Setup complete\r\n\r\n");
    return 0;
}

//send command to Bluetooth and return if there is a response
int sendBlueToothCommand(String command, String expected)
{
    Serial.print("send: ");
    Serial.println(command);
    Serial1.print(command);

    Serial.print("recv: ");
    Serial.println(expectBTResponse(expected, 20000));
    delay(500);
    return 0;
}

//receive message from Bluetooth with time out
String expectBTResponse(String expected, unsigned long timeout)
{
  String response = "";
  unsigned long time = 0;
  while (!response.startsWith(expected) && time<timeout) {
    while (Serial1.available()) {
        char c = Serial1.read();
        response += c;
        if (c == '\n') {
          return response;
        }
    }
    delay(10);
    time += 10;
  }
  return response;
}
