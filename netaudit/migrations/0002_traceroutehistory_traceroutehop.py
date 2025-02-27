# Generated by Django 5.1.4 on 2024-12-25 08:05

import django.db.models.deletion
from django.db import migrations, models


class Migration(migrations.Migration):

    dependencies = [
        ("netaudit", "0001_initial"),
    ]

    operations = [
        migrations.CreateModel(
            name="TracerouteHistory",
            fields=[
                (
                    "id",
                    models.BigAutoField(
                        auto_created=True,
                        primary_key=True,
                        serialize=False,
                        verbose_name="ID",
                    ),
                ),
                ("ip_address", models.CharField(max_length=45)),
                ("timestamp", models.DateTimeField(auto_now_add=True)),
                ("is_successful", models.BooleanField()),
                ("total_hops", models.IntegerField(null=True)),
                ("completion_time", models.FloatField(null=True)),
            ],
            options={
                "ordering": ["-timestamp"],
            },
        ),
        migrations.CreateModel(
            name="TracerouteHop",
            fields=[
                (
                    "id",
                    models.BigAutoField(
                        auto_created=True,
                        primary_key=True,
                        serialize=False,
                        verbose_name="ID",
                    ),
                ),
                ("hop_number", models.IntegerField()),
                ("hostname", models.CharField(blank=True, max_length=255, null=True)),
                ("ip_address", models.CharField(blank=True, max_length=45, null=True)),
                ("rtt1", models.FloatField(null=True)),
                ("rtt2", models.FloatField(null=True)),
                ("rtt3", models.FloatField(null=True)),
                (
                    "traceroute",
                    models.ForeignKey(
                        on_delete=django.db.models.deletion.CASCADE,
                        related_name="hops",
                        to="netaudit.traceroutehistory",
                    ),
                ),
            ],
            options={
                "ordering": ["hop_number"],
            },
        ),
    ]
