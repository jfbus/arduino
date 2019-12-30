#!/usr/bin/python3

# Imports the hardware library
import adafruit_is31fl3731
import board
import busio

import time

import os

# Set the following paths
os.chdir("/home/pi/raspberry/arduino/cloud/pictures")
os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = "/home/pi/raspberry/PemFiles/key.json"
import sys

# Imports the Google Cloud client library
from google.cloud import storage
from google.cloud import vision

from picamera import PiCamera

#Set the following variables
bucket_name = "cocoon-ml-datasets"
project_folder_name = "salle_stockage"

# Folder that contains all your pictures
local_folder_path = "/home/pi/raspberry/arduino/cloud/pictures/"
csv_file = "data.csv"

def write_csv_file(uri, label_name):

    # Instantiates a client
    client = vision.ImageAnnotatorClient()

    image = vision.types.Image()
    image.source.image_uri = uri

    # Using google cloud API
    objects = client.object_localization(image=image).localized_object_annotations
    
    # Creating csv file
    file = open(csv_file, "a")

    # Set google cloud csv field 
    field = "UNASSIGNED"

    # Drawing square 
    for object_ in objects:
        file.write("{},{},{}".format(field, uri, label_name))
        for vertex in object_.bounding_poly.normalized_vertices:
            file.write(",{},{}".format(vertex.x, vertex.y))
        file.write("\n")
        
def upload_file(name):

    # Instantiates a client
    client = storage.Client()
    bucket = client.get_bucket(bucket_name)

    # Upload file
    filename = "%s/%s" % (project_folder_name, name)
    blob = bucket.blob(filename)
    blob.upload_from_filename(local_folder_path + name)

def report(name, label_name, nb_pictures, flash_power):

    # Initialize i2c bus
    i2c = busio.I2C(board.SCL, board.SDA)
    display = adafruit_is31fl3731.CharlieBonnet(i2c)

    # Initialize raspberry camera
    camera = PiCamera()
    
    number = 0
    extension = '.jpg'

    # Start taking pictures
    for i in range (int(nb_pictures)):
        display.fill(flash_power)
        camera.start_preview()
        #time.sleeip(2)
        pictureName = name + str(number) + extension
        camera.capture(pictureName)
        upload_file(pictureName)
        uri = "gs://" + bucket_name + "/" + project_folder_name + "/" + pictureName
        write_csv_file(uri, label_name)
        number = number + 1

    display.fill(0)

    # Upload csv file on google storage
    upload_file(csv_file)
    os.remove(csv_file)

def main():
    report(sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4])


if __name__ == "__main__":
    main()
