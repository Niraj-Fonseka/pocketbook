# Pocketbook

Slack app for storing links,messages or any text you want for quick access.


### Requirements 
---
- [Create a Slack App](https://api.slack.com/apps)
- Set these environment variables
```
SLACK_BOT_TOKEN - bot token from the slack app
 
SLACK_APP_TOKEN - app token from the slack app

GOOGLE_APPLICATION_CREDENTIALS - service account to gain access to firestore
```

### How to use
---

- Add a new note
    - command - `/pocketbook https://news.ycombinator.com/`
    - respone - no response
- List all notes
    - command - `/pocketbook`
    - response 
        -  
- Select a note and send it to the channel
- Delete a note 