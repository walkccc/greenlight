BODY='{"name": "Alice Smith", "email": "alice@example.com", "password": "pa55word"}'
curl -i -d "$BODY" localhost:4000/v1/users

BODY='{"name": "", "email": "bob@invalid.", "password": "pass"}'
curl -d "$BODY" localhost:4000/v1/users

BODY='{"name": "Alice Jones", "email": "alice@example.com", "password": "pa55word"}'
curl -i -d "$BODY" localhost:4000/v1/users

BODY='{"name": "Bob Jones", "email": "bob@example.com", "password": "pa55word"}'
curl -w '\nTime: %{time_total}\n' -d "$BODY" localhost:4000/v1/users

BODY='{"name": "Dave Smith", "email": "dave@example.com", "password": "pa55word"}'
curl -w '\nTime: %{time_total}\n' -d "$BODY" localhost:4000/v1/users

BODY='{"name": "Edith Smith", "email": "edith@example.com", "password": "pa55word"}'
curl -d "$BODY" localhost:4000/v1/users & pkill -SIGTERM api &
