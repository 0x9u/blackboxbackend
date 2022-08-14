# What to do left
- Allow typing states in backend
- Have file support
- Have captcha on backend
- Have games support (single player)
- multiplayer games
- Shorten authentication using middleware
- Make a failsafe for same invite code
- ADd some guild shit (edit settings) 
- Add some tests (Im not bothered to do this so I need to be on something probs)
- Modulize the backend (Not too sure yet how I should handle the global variables)
    - make global variables its own module
        - possible bad practice so research more into it
        - modulize each file and split long lengthy functions into files in a folder

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

**AFTER FINISHED EVERYTHING**
- Implement rate limiting
    - store in ram in a map of ips pointing to timers
    - 500 ms rate limit
- Implement email checking so theres no duplicate emails
