
#define INP0 0
#define INP1 1
#define INP2 2

#define OUT0 2
#define OUT1 3
#define OUT2 4

#define LED 13

#define DELAY 1000

int last0 = -1;
int last1 = -1;
int last2 = -1;
int counter = 0;

void setup()
{
  Serial.begin(9600);
  pinMode(OUT0, OUTPUT);
  pinMode(OUT1, OUTPUT);
  pinMode(OUT2, OUTPUT);
  pinMode(LED, OUTPUT);
}

void loop()
{
  reflect(INP0, OUT0, &last0, 100, 230);
  reflect(INP1, OUT1, &last1, 100, 200);
  reflect(INP2, OUT2, &last2, 100, 300);
  counter++;
  if (counter == DELAY) {
    digitalWrite(LED, HIGH);
  } else if (counter == DELAY*2) {
    counter = 0;
    digitalWrite(LED, LOW);
  }
}

void reflect(int in, int out, int *last, int t1, int t2) {
  int v = analogRead(in);
  int b;
  if (v >= t2) {
    b = 1;
  } else if (v < t1) {
    b = 0;
  } else {
   return;
  }
  if (b != *last) {
    *last = b;
    if (b == 0) {
      digitalWrite(out, LOW);
    } else {
      digitalWrite(out, HIGH);
    }
  }
}
