from concurrent import futures
import logging
import grpc
import torch
from gen_py.model import model_pb2, model_pb2_grpc
from transformers import AutoTokenizer, AutoModelForSequenceClassification

class BertService(model_pb2_grpc.BertServiceServicer):
    def __init__(self):
        self.model_name = "seara/rubert-tiny2-russian-sentiment"
        try: 
            self.tokenizer = AutoTokenizer.from_pretrained(self.model_name)
            self.model = AutoModelForSequenceClassification.from_pretrained(self.model_name)
            self.model.eval()
            print("Model and tokenizer loaded successfully")
        except Exception as e:
            print(f"Error loading model: {str(e)}")
            raise

    def Predict(self, request, context):
        try:
            input_text = request.text

            if not isinstance(input_text, str):
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("Expected UTF-8 string")
                return model_pb2.BertResponse()

            print(f"Received text: {input_text}")
            inputs = self.tokenizer(
                input_text,
                return_tensors="pt",
                padding=True,
                truncation=True,
                max_length=512
            )
            
            # Предсказание
            with torch.no_grad():
                outputs = self.model(**inputs)
            
            # Преобразование в вероятности
            probs = torch.softmax(outputs.logits, dim=-1)[0].tolist()
            
            return model_pb2.BertResponse(logits=probs)
            
        except Exception as e:
            print(f"Prediction error: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(f"Prediction error: {str(e)}")
            return model_pb2.BertResponse()

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    model_pb2_grpc.add_BertServiceServicer_to_server(BertService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("BERT gRPC Server running on port 50051...")
    server.wait_for_termination()

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    serve()