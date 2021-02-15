
#include "limits.h"

#define INPUT 1
int last = 0;
int lowest = INT_MAX;
int highest = 0;
int divf = 0;
int lowT = 0;
int highT = 0;
int current = 0;

void setup()
{
  Serial.begin(9600);
}

void loop()
{
   int v, b;
   v = analogRead(INPUT);
   if (v < lowest) {
      lowest = v;
      recalc();
   }
   if (v > highest) {
      highest = v;
      recalc();
   }
   if (v < lowT) {
    b = 0;
   } else if (v > highT) {
    b = 1;
   } else {
    b = -1;
   }
   if (divf != 0) {
    int count = (v - lowest) / divf;
    Serial.print("|");
    for (int i = 0; i <= count; i++) {
      Serial.print("*");
    }
    for (;count < 10; count++) {
      Serial.print(" ");
    }
    Serial.print("|  ");
   }
   Serial.print("H: ");
   Serial.print(highest);
   Serial.print(" L: ");
   Serial.print(lowest);
   Serial.print(" bit: ");
   if (b == -1) {
    Serial.print("?");
   } else {
    Serial.print(b);
    if (current != b) {
      current = b;
      Serial.print(" threshold change ");
    }
   }
   Serial.print(" V: ");
   Serial.println(v);
   delay(10);
}

void recalc() {
  int s = highest - lowest;
  divf = s / 10;
  lowT = lowest + s/3;
  highT = lowest + s * 2 / 3;
}
