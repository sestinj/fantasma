import sys
import json

contents = open(sys.argv[1], 'r').read()
json = json.loads(contents)

print("Hello, " + json['name'] + "!")