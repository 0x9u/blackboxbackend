# What to do left
- Allow typing states in backend - done
- Add pinging - done
- Add notifications - done
- Add clear messages button - done
- Add clear messages from specific user button (put it in settings) - done
- clear messages on save chat off - done
- Add delete account - done
- Add badges for users

- Remove 6 character limit - done
- Have file support - done
    - add file deletion for messages - done
    - add files for user and server profiles - done
    - add compression (LZ4) - done
    - schedule deleton for temp files - done
- Have captcha on backend
- Have games support (single player)
- multiplayer games
- Shorten authentication using middleware - done
- Make a failsafe for same invite code
- ADd some guild shit (edit settings) - done
- Add some tests (Im not bothered to do this so I need to be on something probs)
- Modulize the backend (Not too sure yet how I should handle the global variables) - done
    - make global variables its own module - cancelled
        - possible bad practice so research more into it
        - modulize each file and split long lengthy functions into files in a folder

- Add direct messages
    - Make new tables - done
- add friend requests - done
- persist typing state for guild and users

- add blocked users - done
    - block msg from blocked users including dms

- add friending restrictions - sorta
    - the most fucked - mainly the sql shit
    - might skip and put in v2.0 instead

- add give admin for guilds - done
- add mentioning in msgs - sorta
    - remove duplicate mentions in one msg
    - show total mentions in unread - done

- redo files entity table - done
    - make it reference its own entities instead for better management - Cancelled
        - made it its own table instead

- add system messages
    - join, leave messages


- THERE MAY BE A RACE CONDITION WITH GUILD PROFILE PICTURES AS ADMINS CAN CHANGE CONCURRENTLY SOMEHOW

- fix time stamp issue for json  use ISO 8601 - done

- **IMPORTANT**
    - Encrypt all data somehow in a way that it cannot be decrypted without user
        - idk how this will work but i have to do this for privacy shit
    - make sure its timestamp with time zone - done
        - without time zone creates weird bugs
    - add support for streaming videos somehow
    - fix websocket bugs 
    - add octal descriminators
    - fix edit and delete for unsaved messages
    - fix date for unsaved messages
    - make sure save chat works with files
    - clean up api and use chatgpt to make documentation (LAST)
        - perhaps make own api for server in python
    - fix unread msgs not accurate when reopening dm
    - fix new line msg
    - add websocket delay when client connects
    - Migrate to GORM (LOWEST PRIORITY)
    

v1.0

- show member count

- display global server announcements




v2.0

- add channels (most difficult)

- add permissions for guilds

- clean up code and put it in functions

- add join/left messages

- add reactions

- add replys

- add friend restrictions

- add public guilds
    - sort by member count or new
        - add guild bumping
            - sort by timestamps

# PRIORITY
- Clean up sql database
  - Rename tables to fit standard e.g roles -> role, userguilds -> user_has_guild
- CONVERT THIS INTO GIN (ASAP) - done
  - mux is archived so for maintainbility purposes this will have to be migrated into the gin lib
  - estimated probs 1-2 hours (ended as 4 hours)
- combine multiple queries together and create auto clean up if there is a failure - DONe
    - ADD TRANSACTIONS FOR SQL QUERIES - DONE
    - FIX rollback error bug
        - I accidently put the error in the condition scope 

# MAYBES
- Learn C# and program in unity in WASM (make games)
- Make multiplayer games
- Add machine learning

# Currently doing
- **!!CRITICAL!!** Something is causing high cpu usage when websockets are running - FIXED
    - possibly for loop?
- **!!CRITICAL!!** If the same user were to open multiple clients the client map will be overwritten therefore
    only one of the clients will be able to get the data - (Possibly fixed needs further testing)
- **!!CRITICAL!!** WEIRD WEBSOCKET BUG
    - The websocket handler will not work randomly
    - Most likely is the mutex lock which is causing this
- implement websocket pool for better efficency - done
    - gather websockets associated with a guild - done (Needs more testing)
- Restructure the token to use sql database instead of ram - done
    - make a monthly checkup to remove expired tokens
    - encrypt the tokens with sha256 no salt no pepper since tokens are random af - Cancelled
- 
**AFTER FINISHED EVERYTHING**
- Implement rate limiting - implementing it done sorta havent tested
    - store in ram in a map of ips pointing to timers
    - 500 ms rate limit - done
- Implement email checking so theres no duplicate emails
