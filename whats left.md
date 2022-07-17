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
- **!!CRITICAL!!** Something is causing high cpu usage when websockets are running 
    - possibly for loop?
- implement websocket pool for better efficency - done
    - gather websockets associated with a guild - done (Needs more testing)
- Restructure the token to use sql database instead of ram - done
    - make a monthly checkup to remove expired tokens
    - encrypt the tokens with sha256 no salt no pepper since tokens are random af - Cancelled
- Implement rate limiting
    - store in ram in a map of ips pointing to timers
    - 500 ms rate limit
- Implement email checking so theres no duplicate emails
