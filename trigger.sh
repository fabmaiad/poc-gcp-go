curl -m 70 -X POST https://us-central1-bullla-one-d-apps-cn-92c3.cloudfunctions.net/func-publisher-golang \
-H "Authorization: bearer $(gcloud auth print-identity-token)" \
-H "Content-Type: application/json" \
-d '{
  "name": "Hello World"
}'