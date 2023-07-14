# "Find My" Tail

Use your iCloud session from Chrome to get a stream of JSON location information from
the Apple's "Find My" application.

```bash
 go run . | jq  '.content[0].location'
 ```
 ```json
 {
  "isOld": false,
  "isInaccurate": false,
  "altitude": 0,
  "addresses": null,
  "positionType": "Unknown",
  "secureLocation": null,
  "secureLocationTs": 0,
  "latitude": 40.70,
  "floorLevel": 0,
  "horizontalAccuracy": 7.266115350618051,
  "locationType": "",
  "timeStamp": 168936400000,
  "locationFinished": true,
  "verticalAccuracy": 0,
  "locationMode": null,
  "longitude": -74.0
}
```

# Bugs

1. Apple prompts you to log in all the time. *Working on it!*
2. There's no way to use an existing Chrome session. So we copy cookies from your existing Chrome profile over. This will prompt you for a Keychain password on Mac!