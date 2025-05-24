from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Iterable as _Iterable, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class BertRequest(_message.Message):
    __slots__ = ("text",)
    TEXT_FIELD_NUMBER: _ClassVar[int]
    text: str
    def __init__(self, text: _Optional[str] = ...) -> None: ...

class BertResponse(_message.Message):
    __slots__ = ("logits",)
    LOGITS_FIELD_NUMBER: _ClassVar[int]
    logits: _containers.RepeatedScalarFieldContainer[float]
    def __init__(self, logits: _Optional[_Iterable[float]] = ...) -> None: ...
